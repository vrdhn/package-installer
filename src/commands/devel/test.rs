use log::{info, error};
use std::path::Path;
use regex::Regex;
use crate::starlark::runtime::{evaluate_file, execute_function};
use crate::models::package_entry::PackageEntry;

pub fn run(filename: &str, pkg: Option<&str>) {
    info!("Executing devel test command for file: {}", filename);
    
    let path = Path::new(filename);
    match evaluate_file(path) {
        Ok(packages) => {
            info!("Registered {} packages.", packages.len());
            if let Some(package_name) = pkg {
                process_package_matching(package_name, &packages);
            }
        }
        Err(e) => error!("Starlark evaluation failed: {}", e),
    }
}

fn process_package_matching(package_name: &str, packages: &[PackageEntry]) {
    for entry in packages {
        match Regex::new(&entry.regexp) {
            Ok(re) => {
                if re.is_match(package_name) {
                    run_package_function(package_name, entry);
                }
            }
            Err(e) => error!("Invalid regex '{}': {}", entry.regexp, e),
        }
    }
}

fn run_package_function(package_name: &str, entry: &PackageEntry) {
    info!("Package '{}' matched regex '{}'. Calling function '{}' from '{}'.", 
        package_name, entry.regexp, entry.function_name, entry.filename);
    
    let starlark_path = Path::new(&entry.filename);
    if let Err(e) = execute_function(starlark_path, &entry.function_name, package_name) {
        error!("Function execution failed: {}", e);
    }
}
