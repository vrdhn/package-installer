use starlark::environment::{GlobalsBuilder, Module, LibraryExtension};
use starlark::eval::Evaluator;
use starlark::syntax::{AstModule, Dialect};
use starlark::values::ValueLike;
use std::path::Path;
use std::fs;
use anyhow::Context as _;
use crate::config::{Context, starlark_functions, PackageEntry};

pub fn evaluate_file(path: &Path, package_list: &mut Vec<PackageEntry>) -> anyhow::Result<()> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;
    
    let ast = AstModule::parse(
        &filename,
        content,
        &Dialect::Extended,
    ).map_err(|e| anyhow::anyhow!("{}", e))?;

    let mut globals_builder = GlobalsBuilder::extended_by(&[
        LibraryExtension::Print,
    ]);
    
    starlark_functions(&mut globals_builder);
    let globals = globals_builder.build();
    
    let module = Module::new();
    let context = Context::new(filename);
    
    let context_value = module.heap().alloc_simple(context);
    module.set_extra_value(context_value);
    
    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals).map_err(|e| anyhow::anyhow!("{}", e))?;
    
    let final_context = module.extra_value()
        .context("Context missing after evaluation")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")?;
    
    package_list.extend(final_context.packages.read().clone());
    
    Ok(())
}

pub fn execute_function(path: &Path, function_name: &str, package_name: &str) -> anyhow::Result<()> {
    let filename = path.to_string_lossy().into_owned();
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;
    
    let ast = AstModule::parse(
        &filename,
        content,
        &Dialect::Extended,
    ).map_err(|e| anyhow::anyhow!("{}", e))?;

    let mut globals_builder = GlobalsBuilder::extended_by(&[
        LibraryExtension::Print,
    ]);
    
    starlark_functions(&mut globals_builder);
    let globals = globals_builder.build();
    
    let module = Module::new();
    // Use a dummy filename for the context during function execution if add_package is called
    let context = Context::new(format!("{}:exec", filename));
    
    let context_value = module.heap().alloc_simple(context);
    module.set_extra_value(context_value);
    
    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals).map_err(|e| anyhow::anyhow!("{}", e))?;
    
    let function = module.get(function_name)
        .context(format!("Function '{}' not found in module '{}'", function_name, filename))?;
    
    let pkg_value = eval.heap().alloc(package_name);
    eval.eval_function(function, &[pkg_value], &[])
        .map_err(|e| anyhow::anyhow!("{}", e))?;
    
    Ok(())
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
        
        let mut packages = Vec::new();
        evaluate_file(file.path(), &mut packages).unwrap();
        assert_eq!(packages.len(), 1);
        assert_eq!(packages[0].regexp, "^vlc");
        
        execute_function(file.path(), &packages[0].function_name, "vlc-player").unwrap();
    }
}