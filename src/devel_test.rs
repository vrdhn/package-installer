use log::{info, trace, error};
use std::path::Path;
use crate::starlark_executor;
use regex::Regex;

pub fn run(filename: &str, pkg: Option<&str>) {
    info!("Executing devel test command");
    trace!("Debugging information: filename={}, pkg={:?}", filename, pkg);
    
    let mut package_list = Vec::new();
    let path = Path::new(filename);

    if let Err(e) = starlark_executor::evaluate_file(path, &mut package_list) {
        error!("Starlark evaluation failed: {}", e);
        return;
    }

    info!("Registered {} packages.", package_list.len());

    if let Some(package_name) = pkg {
        for entry in package_list {
            match Regex::new(&entry.regexp) {
                Ok(re) => {
                    if re.is_match(package_name) {
                        info!("Package '{}' matched regex '{}'. Calling function '{}' from '{}'.", 
                            package_name, entry.regexp, entry.function_name, entry.filename);
                        
                        let starlark_path = Path::new(&entry.filename);
                        if let Err(e) = starlark_executor::execute_function(
                            starlark_path, 
                            &entry.function_name, 
                            package_name
                        ) {
                            error!("Function execution failed: {}", e);
                        }
                    }
                }
                Err(e) => {
                    error!("Invalid regex '{}': {}", entry.regexp, e);
                }
            }
        }
    }
    
    info!("Devel test command completed");
}
