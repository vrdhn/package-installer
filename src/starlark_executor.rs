use starlark::environment::{GlobalsBuilder, Module, LibraryExtension};
use starlark::eval::Evaluator;
use starlark::syntax::{AstModule, Dialect};
use starlark::values::ValueLike;
use std::path::Path;
use std::fs;
use anyhow::Context as _;
use crate::config::{Context, starlark_functions};
use regex::Regex;

pub fn evaluate_file(path: &Path, pkg: Option<&str>) -> anyhow::Result<()> {
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;
    
    let ast = AstModule::parse(
        &path.to_string_lossy(),
        content,
        &Dialect::Extended,
    ).map_err(|e| anyhow::anyhow!("{}", e))?;

    let mut globals_builder = GlobalsBuilder::extended_by(&[
        LibraryExtension::Print,
    ]);
    
    // Register starlark functions
    starlark_functions(&mut globals_builder);
    let globals = globals_builder.build();
    
    let module = Module::new();
    let context = Context::new();
    
    let context_value = module.heap().alloc_simple(context);
    module.set_extra_value(context_value);
    
    let mut eval = Evaluator::new(&module);
    eval.eval_module(ast, &globals).map_err(|e| anyhow::anyhow!("{}", e))?;
    
    if let Some(package_name) = pkg {
        let final_context = module.extra_value()
            .context("Context missing after evaluation")?
            .downcast_ref::<Context>()
            .context("Extra value is not a Context")?;
        
        let packages = final_context.packages.read();
        for entry in packages.iter() {
            let re = Regex::new(&entry.regexp)
                .with_context(|| format!("Invalid regex: {}", entry.regexp))?;
            
            if re.is_match(package_name) {
                log::info!("Package '{}' matched regex '{}'. Calling function '{}'.", package_name, entry.regexp, entry.function_name);
                
                // Look up the function in the module
                let function = module.get(&entry.function_name)
                    .context(format!("Function '{}' not found in module", entry.function_name))?;
                
                let pkg_value = eval.heap().alloc(package_name);
                eval.eval_function(function, &[pkg_value], &[])
                    .map_err(|e| anyhow::anyhow!("{}", e))?;
            }
        }
    }
    
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_add_package_and_call() {
        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "def install_vlc(pkg): print('Installing', pkg)").unwrap();
        writeln!(file, "add_package('^vlc', install_vlc)").unwrap();
        
        let result = evaluate_file(file.path(), Some("vlc-player"));
        assert!(result.is_ok());
    }
}
