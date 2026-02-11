use log::{info, error};
use std::path::{Path, PathBuf};
use regex::Regex;
use crate::starlark::runtime::{evaluate_file, execute_function};
use crate::models::package_entry::PackageEntry;

pub fn run(filename: &str, pkg: Option<&str>) {
    info!("Executing devel test command for file: {}", filename);
    
    // Default download directory for test
    let download_dir = get_default_download_dir();
    
    let path = Path::new(filename);
    match evaluate_file(path, download_dir.clone()) {
        Ok(packages) => {
            info!("Registered {} packages.", packages.len());
            if let Some(package_name) = pkg {
                process_package_matching(package_name, &packages, download_dir);
            }
        }
        Err(e) => error!("Starlark evaluation failed: {}", e),
    }
}

fn get_default_download_dir() -> PathBuf {
    let mut path = dirs_next::cache_dir().unwrap_or_else(|| PathBuf::from(".cache"));
    path.push("pi");
    path.push("downloads");
    path
}

fn process_package_matching(package_name: &str, packages: &[PackageEntry], download_dir: PathBuf) {
    for entry in packages {
        match Regex::new(&entry.regexp) {
            Ok(re) => {
                if re.is_match(package_name) {
                    run_package_function(package_name, entry, download_dir.clone());
                }
            }
            Err(e) => error!("Invalid regex '{}': {}", entry.regexp, e),
        }
    }
}

fn run_package_function(package_name: &str, entry: &PackageEntry, download_dir: PathBuf) {
    info!("Package '{}' matched regex '{}'. Calling function '{}' from '{}'.", 
        package_name, entry.regexp, entry.function_name, entry.filename);
    
    let starlark_path = Path::new(&entry.filename);
    if let Err(e) = execute_function(starlark_path, &entry.function_name, package_name, download_dir) {
        error!("Function execution failed: {}", e);
    }
}