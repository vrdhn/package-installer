use crate::models::config::Config;
use crate::models::package_entry::PackageEntry;
use crate::models::version_entry::VersionEntry;
use crate::starlark::runtime::{evaluate_file, execute_function};
use comfy_table::presets::NOTHING;
use comfy_table::Table;
use log::{error, info};
use std::path::Path;

pub fn run(config: &Config, filename: &str, pkg: Option<&str>) {
    info!("Executing devel test command for file: {}", filename);

    let download_dir = config.download_dir.clone();

    let path = Path::new(filename);
    match evaluate_file(path, download_dir.clone(), config.state.clone()) {
        Ok((packages, managers)) => {
            info!("Registered {} packages and {} managers.", packages.len(), managers.len());
            if let Some(package_name) = pkg {
                // Try manager first if it's a manager:package format
                if let Some(colon_idx) = package_name.find(':') {
                    let mgr_name = &package_name[..colon_idx];
                    let pkg_inner = &package_name[colon_idx + 1..];

                    if let Some(mgr) = managers.iter().find(|m| m.name == mgr_name) {
                        run_manager_function(config, mgr_name, pkg_inner, mgr);
                        return;
                    }
                }

                // Try exact package name match
                if let Some(pkg_entry) = packages.iter().find(|p| p.name == package_name) {
                    run_package_function(config, package_name, pkg_entry);
                    return;
                }

                error!("Package or manager '{}' not found in file.", package_name);
            }
        }
        Err(e) => error!("Starlark evaluation failed: {}", e),
    }
}

fn run_manager_function(config: &Config, manager_name: &str, package_name: &str, entry: &crate::models::package_entry::ManagerEntry) {
    info!(
        "Manager '{}' matched exactly. Calling function '{}' for package '{}' from '{}'.",
        manager_name, entry.function_name, package_name, entry.filename
    );

    let download_dir = config.download_dir.clone();
    let starlark_path = Path::new(&entry.filename);
    match crate::starlark::runtime::execute_manager_function(
        starlark_path,
        &entry.function_name,
        manager_name,
        package_name,
        download_dir,
        config.state.clone(),
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
        Err(e) => error!("Manager function execution failed: {}", e),
    }
}

fn run_package_function(config: &Config, package_name: &str, entry: &PackageEntry) {
    info!(
        "Package '{}' matched exactly. Calling function '{}' from '{}'.",
        package_name, entry.function_name, entry.filename
    );

    let download_dir = config.download_dir.clone();
    let starlark_path = Path::new(&entry.filename);
    match execute_function(
        starlark_path,
        &entry.function_name,
        package_name,
        download_dir,
        config.state.clone(),
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
    table.load_preset(NOTHING);
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
