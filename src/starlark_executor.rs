use starlark::environment::{Globals, Module, LibraryExtension};
use starlark::eval::Evaluator;
use starlark::syntax::{AstModule, Dialect};
use std::path::Path;
use std::fs;
use anyhow::Context;

pub fn evaluate_file(path: &Path) -> anyhow::Result<()> {
    let content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read file: {}", path.display()))?;
    
    let ast = AstModule::parse(
        &path.to_string_lossy(),
        content,
        &Dialect::Extended,
    ).map_err(|e| anyhow::anyhow!("{}", e))?;

    let globals = Globals::extended_by(&[
        LibraryExtension::Print,
    ]);
    let module = Module::new();
    let mut eval = Evaluator::new(&module);
    
    eval.eval_module(ast, &globals).map_err(|e| anyhow::anyhow!("{}", e))?;
    
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_evaluate_file() {
        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "print('test')").unwrap();
        
        let result = evaluate_file(file.path());
        assert!(result.is_ok());
    }
}

