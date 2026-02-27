use crate::models::config::State;
use crate::models::context::Context;
use crate::models::package_entry::{ManagerEntry, PackageEntry};
use crate::models::version_entry::VersionEntry;
use crate::starlark::api::register_api;
use anyhow::Context as _;
use starlark::analysis::AstModuleLint;
use starlark::environment::{GlobalsBuilder, LibraryExtension, Module};
use starlark::eval::Evaluator;
use starlark::syntax::{AstModule, Dialect};
use starlark::values::ValueLike;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::Arc;

pub fn evaluate_file(
    path: &Path,
    state: Arc<State>,
) -> anyhow::Result<(Vec<PackageEntry>, Vec<ManagerEntry>)> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;

    let ast = parse_ast(&filename, content)?;
    lint_ast(&filename, &ast);

    let globals = create_globals();
    let module = Module::new();

    setup_context(
        &module,
        filename,
        state.meta_dir.clone(),
        state.download_dir.clone(),
        state.packages_dir.clone(),
        state.clone(),
        None,
    );

    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals)
        .map_err(|e| anyhow::anyhow!("{}", e))?;

    let packages = extract_packages(&module)?;
    let managers = extract_managers(&module)?;
    Ok((packages, managers))
}

pub fn execute_manager_function(
    path: &Path,
    function_name: &str,
    manager_name: &str,
    package_name: &str,
    state: Arc<State>,
    options: Option<std::collections::HashMap<String, String>>,
) -> anyhow::Result<Vec<VersionEntry>> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;

    let ast = parse_ast(&filename, content)?;
    lint_ast(&filename, &ast);

    let globals = create_globals();
    let module = Module::new();

    setup_context(
        &module,
        format!("{}:exec:{}", filename, manager_name),
        state.meta_dir.clone(),
        state.download_dir.clone(),
        state.packages_dir.clone(),
        state.clone(),
        options,
    );

    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals)
        .map_err(|e| anyhow::anyhow!("{}", e))?;

    let function = module.get(function_name).context(format!(
        "Function '{}' not found in module '{}'",
        function_name, filename
    ))?;

    let mgr_val = eval.heap().alloc(manager_name);
    let pkg_val = eval.heap().alloc(package_name);
    eval.eval_function(function, &[mgr_val, pkg_val], &[])
        .map_err(|e| anyhow::anyhow!("{}", e))?;

    extract_versions(&module)
}

pub fn execute_function(
    path: &Path,
    function_name: &str,
    argument: &str,
    state: Arc<State>,
    options: Option<std::collections::HashMap<String, String>>,
) -> anyhow::Result<Vec<VersionEntry>> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;

    let ast = parse_ast(&filename, content)?;
    lint_ast(&filename, &ast);

    let globals = create_globals();
    let module = Module::new();

    setup_context(
        &module,
        format!("{}:exec", filename),
        state.meta_dir.clone(),
        state.download_dir.clone(),
        state.packages_dir.clone(),
        state.clone(),
        options,
    );

    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals)
        .map_err(|e| anyhow::anyhow!("{}", e))?;

    let function = module.get(function_name).context(format!(
        "Function '{}' not found in module '{}'",
        function_name, filename
    ))?;

    let arg_value = eval.heap().alloc(argument);
    eval.eval_function(function, &[arg_value], &[])
        .map_err(|e| anyhow::anyhow!("{}", e))?;

    extract_versions(&module)
}

fn parse_ast(filename: &str, content: String) -> anyhow::Result<AstModule> {
    AstModule::parse(filename, content, &Dialect::Extended).map_err(|e| anyhow::anyhow!("{}", e))
}

fn lint_ast(filename: &str, ast: &AstModule) {
    let globals = create_globals();
    let names: std::collections::HashSet<String> = globals.names().map(|s| s.as_str().to_string()).collect();
    for lint in ast.lint(Some(&names)) {
        log::warn!("[{}] lint: {} ({})", filename, lint.problem, lint.location);
    }
}

fn create_globals() -> starlark::environment::Globals {
    let mut builder =
        GlobalsBuilder::extended_by(&[LibraryExtension::Print, LibraryExtension::Json]);
    register_api(&mut builder);
    builder.build()
}

fn setup_context(
    module: &Module,
    filename: String,
    meta_dir: PathBuf,
    download_dir: PathBuf,
    packages_dir: PathBuf,
    state: Arc<State>,
    options: Option<std::collections::HashMap<String, String>>,
) {
    let mut context = Context::new(filename, meta_dir, download_dir, packages_dir, state);
    if let Some(opts) = options {
        context = context.with_options(opts);
    }
    let context_value = module.heap().alloc_simple(context);
    module.set_extra_value(context_value);
}

fn extract_packages(module: &Module) -> anyhow::Result<Vec<PackageEntry>> {
    let context = module
        .extra_value()
        .context("Context missing after evaluation")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")?;

    Ok(context.packages.read().clone())
}

pub fn extract_managers(module: &Module) -> anyhow::Result<Vec<ManagerEntry>> {
    let context = module
        .extra_value()
        .context("Context missing after evaluation")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")?;

    Ok(context.managers.read().clone())
}

fn extract_versions(module: &Module) -> anyhow::Result<Vec<VersionEntry>> {
    let context = module
        .extra_value()
        .context("Context missing after evaluation")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")?;

    Ok(context.versions.read().clone())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_evaluate_and_execute() {
        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "def install_vlc(pkg): print('Installing', pkg)").unwrap();
        writeln!(file, "add_package('^vlc', install_vlc)").unwrap();

        let meta_dir = PathBuf::from("/tmp/pi-test-meta");
        let download_dir = PathBuf::from("/tmp/pi-test-downloads");
        let packages_dir = PathBuf::from("/tmp/pi-test-packages");
        let state = Arc::new(State {
            meta_dir,
            download_dir,
            packages_dir,
            ..Default::default()
        });
        let (packages, _managers) = evaluate_file(file.path(), state.clone()).unwrap();
        assert_eq!(packages.len(), 1);
        assert_eq!(packages[0].name, "^vlc");

        let versions = execute_function(
            file.path(),
            &packages[0].function_name,
            "vlc-player",
            state,
            None,
        )
        .unwrap();
        assert_eq!(versions.len(), 0);
    }

    #[test]
    fn test_extract() {
        let meta_dir = PathBuf::from("/tmp/pi-test-meta-re");
        let download_dir = PathBuf::from("/tmp/pi-test-downloads-re");
        let packages_dir = PathBuf::from("/tmp/pi-test-packages-re");
        let state = Arc::new(State {
            meta_dir,
            download_dir,
            packages_dir,
            ..Default::default()
        });

        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "def test(arg):").unwrap();
        writeln!(file, "    ok, name, version = extract(r'([a-z]+)-([0-9.]+)', 'python-3.9')").unwrap();
        writeln!(file, "    if not ok or name != 'python' or version != '3.9':").unwrap();
        writeln!(file, "        fail('Match failed: ' + str(ok) + ' ' + name + ' ' + version)").unwrap();
        writeln!(file, "    ok2, g1 = extract(r'(abc)', 'def')").unwrap();
        writeln!(file, "    if ok2:").unwrap();
        writeln!(file, "        fail('Should not match')").unwrap();
        writeln!(file, "    if g1 != '':").unwrap();
        writeln!(file, "        fail('Group should be empty')").unwrap();
        writeln!(file, "add_package('test', test)").unwrap();

        let (packages, _) = evaluate_file(file.path(), state.clone()).unwrap();
        execute_function(file.path(), &packages[0].function_name, "", state, None).unwrap();
    }

    #[test]
    fn test_datanode_get_default() {
        let meta_dir = PathBuf::from("/tmp/pi-test-meta-get-default");
        let download_dir = PathBuf::from("/tmp/pi-test-downloads-get-default");
        let packages_dir = PathBuf::from("/tmp/pi-test-packages-get-default");
        let state = Arc::new(State {
            meta_dir,
            download_dir,
            packages_dir,
            ..Default::default()
        });

        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "def test(arg):").unwrap();
        writeln!(file, r#"    doc = parse_json('{{ "a": 1 }}')"#).unwrap();
        writeln!(file, "    data = doc.root").unwrap();
        writeln!(file, r#"    val = data.get("b", "default_val")"#).unwrap();
        writeln!(file, r#"    if val != "default_val": fail("Expected default_val, got " + str(val))"#).unwrap();
        writeln!(file, r#"    val_existing = data.get("a", "default_val")"#).unwrap();
        writeln!(file, r#"    if val_existing != 1: fail("Expected 1, got " + str(val_existing))"#).unwrap();
        writeln!(file, "add_package('test', test)").unwrap();

        let (packages, _) = evaluate_file(file.path(), state.clone()).unwrap();
        execute_function(file.path(), &packages[0].function_name, "", state, None).unwrap();
    }

    #[test]
    fn test_datanode_iteration() {
        let meta_dir = PathBuf::from("/tmp/pi-test-meta-datanode");
        let download_dir = PathBuf::from("/tmp/pi-test-downloads-datanode");
        let packages_dir = PathBuf::from("/tmp/pi-test-packages-datanode");
        let state = Arc::new(State {
            meta_dir,
            download_dir,
            packages_dir,
            ..Default::default()
        });

        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "def test(arg):").unwrap();
        writeln!(file, r#"    doc = parse_json('[{{ "v": "1.0" }}, {{ "v": "2.0" }}]')"#).unwrap();
        writeln!(file, "    data = doc.root").unwrap();
        writeln!(file, "    count = 0").unwrap();
        writeln!(file, "    for item in data:").unwrap();
        writeln!(file, "        count += 1").unwrap();
        writeln!(file, "        v = item.get(\"v\")").unwrap();
        writeln!(file, "        if count == 1 and v != \"1.0\": fail(\"Expected 1.0\")").unwrap();
        writeln!(file, "        if count == 2 and v != \"2.0\": fail(\"Expected 2.0\")").unwrap();
        writeln!(file, "    if count != 2: fail(\"Expected 2 items, got \" + str(count))").unwrap();
        writeln!(file, "add_package('test', test)").unwrap();

        let (packages, _) = evaluate_file(file.path(), state.clone()).unwrap();
        execute_function(file.path(), &packages[0].function_name, "", state, None).unwrap();
    }
}
