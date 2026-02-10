use starlark::environment::{GlobalsBuilder, Module, LibraryExtension};
use starlark::eval::Evaluator;
use starlark::syntax::{AstModule, Dialect};
use starlark::values::ValueLike;
use std::path::Path;
use std::fs;
use anyhow::Context as _;
use crate::config::{Context, starlark_functions};

pub fn evaluate_file(path: &Path) -> anyhow::Result<()> {
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
    
    let final_context = module.extra_value()
        .context("Context missing after evaluation")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")?;
    
    let packages = final_context.packages.read();
    log::info!("Evaluation finished. Registered {} packages.", packages.len());
    for p in packages.iter() {
        log::info!("Package: regexp='{}', function='{}'", p.regexp, p.function_name);
    }
    
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_add_package_function() {
        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "def install_vlc(pkg): print('Installing', pkg)").unwrap();
        writeln!(file, "add_package('^vLC', install_vlc)").unwrap();
        
        let result = evaluate_file(file.path());
        assert!(result.is_ok());
    }
}
