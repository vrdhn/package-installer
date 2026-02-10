use log::{info, trace, error};
use std::path::Path;
use crate::starlark_executor;

pub fn run(filename: &str, pkg: Option<&str>) {
    info!("Executing devel test command");
    trace!("Debugging information: filename={}, pkg={:?}", filename, pkg);
    
    if let Err(e) = execute_starlark(filename, pkg) {
        error!("Starlark execution failed: {}", e);
    }
    
    info!("Devel test command completed");
}

fn execute_starlark(filename: &str, pkg: Option<&str>) -> anyhow::Result<()> {
    let path = Path::new(filename);
    starlark_executor::evaluate_file(path, pkg)
}