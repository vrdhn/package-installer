use crate::models::package_entry::PackageEntry;
use crate::models::version_entry::VersionEntry;
use crate::starlark::runtime::{evaluate_file, execute_function};
use comfy_table::Table;
use log::{error, info};
use regex::Regex;
use std::path::{Path, PathBuf};

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
        match Regex::new(&entry.name) {
            Ok(re) => {
                if re.is_match(package_name) {
                    run_package_function(package_name, entry, download_dir.clone());
                }
            }
            Err(e) => error!("Invalid regex '{}': {}", entry.name, e),
        }
    }
}

fn run_package_function(package_name: &str, entry: &PackageEntry, download_dir: PathBuf) {
    info!(
        "Package '{}' matched regex '{}'. Calling function '{}' from '{}'.",
        package_name, entry.name, entry.function_name, entry.filename
    );

    let starlark_path = Path::new(&entry.filename);
    match execute_function(
        starlark_path,
        &entry.function_name,
        package_name,
        download_dir,
    ) {
        Ok(mut versions) => {
            info!(
                "Function execution finished. Found {} versions.",
                versions.len()
            );

            // Sort by date then by version
            versions.sort_by(|a, b| {
                a.release_date
                    .cmp(&b.release_date)
                    .then_with(|| a.version.cmp(&b.version))
            });

            print_versions_table(&versions);
        }
        Err(e) => error!("Function execution failed: {}", e),
    }
}

fn print_versions_table(versions: &[VersionEntry]) {
    if versions.is_empty() {
        return;
    }

    let mut table = Table::new();
    table.set_header(vec![
        "Package",
        "Version",
        "Release Date",
        "Type",
        "Filename",
    ]);

    for v in versions {
        table.add_row(vec![
            &v.pkgname,
            &v.version,
            &v.release_date,
            &v.release_type,
            &v.filename,
        ]);
    }

    println!("{}", table);
}
