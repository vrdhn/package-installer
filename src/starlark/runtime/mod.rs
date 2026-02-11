use starlark::environment::{GlobalsBuilder, Module, LibraryExtension};
use starlark::eval::Evaluator;
use starlark::syntax::{AstModule, Dialect};
use starlark::values::ValueLike;
use std::path::{Path, PathBuf};
use std::fs;
use anyhow::Context as _;
use crate::models::package_entry::PackageEntry;
use crate::models::context::Context;
use crate::starlark::api::register_api;

pub fn evaluate_file(path: &Path, download_dir: PathBuf) -> anyhow::Result<Vec<PackageEntry>> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;
    
    let ast = parse_ast(&filename, content)?;
    let globals = create_globals();
    let module = Module::new();
    
    setup_context(&module, filename, download_dir);
    
    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals).map_err(|e| anyhow::anyhow!("{}", e))?;
    
    extract_packages(&module)
}

pub fn execute_function(path: &Path, function_name: &str, argument: &str, download_dir: PathBuf) -> anyhow::Result<()> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;
    
    let ast = parse_ast(&filename, content)?;
    let globals = create_globals();
    let module = Module::new();
    
    setup_context(&module, format!("{}:exec", filename), download_dir);
    
    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals).map_err(|e| anyhow::anyhow!("{}", e))?;
    
    let function = module.get(function_name)
        .context(format!("Function '{}' not found in module '{}'", function_name, filename))?;
    
    let arg_value = eval.heap().alloc(argument);
    eval.eval_function(function, &[arg_value], &[])
        .map_err(|e| anyhow::anyhow!("{}", e))?;
    
    Ok(())
}

fn parse_ast(filename: &str, content: String) -> anyhow::Result<AstModule> {
    AstModule::parse(filename, content, &Dialect::Extended)
        .map_err(|e| anyhow::anyhow!("{}", e))
}

fn create_globals() -> starlark::environment::Globals {
    let mut builder = GlobalsBuilder::extended_by(&[
        LibraryExtension::Print,
        LibraryExtension::Json,
    ]);
    register_api(&mut builder);
    builder.build()
}

fn setup_context(module: &Module, filename: String, download_dir: PathBuf) {
    let context = Context::new(filename, download_dir);
    let context_value = module.heap().alloc_simple(context);
    module.set_extra_value(context_value);
}

fn extract_packages(module: &Module) -> anyhow::Result<Vec<PackageEntry>> {
    let context = module.extra_value()
        .context("Context missing after evaluation")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")?;
    
    Ok(context.packages.read().clone())
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
        
        let download_dir = PathBuf::from("/tmp/pi-test");
        let packages = evaluate_file(file.path(), download_dir.clone()).unwrap();
        assert_eq!(packages.len(), 1);
        assert_eq!(packages[0].regexp, "^vlc");
        
        execute_function(file.path(), &packages[0].function_name, "vlc-player", download_dir).unwrap();
    }
}
