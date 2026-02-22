use crate::models::config::Config;
use crate::models::version_entry::VersionEntry;
use log::{error, info};
use std::path::Path;

pub fn run(config: &Config, filename: &str, pkg: Option<&str>) {
    info!("testing file: {}", filename);

    let path = Path::new(filename);
    match crate::starlark::runtime::evaluate_file(path, config.state.clone()) {
        Ok((packages, managers)) => {
            info!("registered {} pkgs, {} mgrs", packages.len(), managers.len());
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

                error!("pkg/mgr {} not found", package_name);
            }
        }
        Err(e) => error!("eval failed: {}", e),
    }
}

fn run_manager_function(config: &Config, manager_name: &str, package_name: &str, entry: &crate::models::package_entry::ManagerEntry) {
    info!(
        "matched mgr: {} calling {} for {} in {}",
        manager_name, entry.function_name, package_name, entry.filename
    );

    let star_path = Path::new(&entry.filename);
    match crate::starlark::runtime::execute_manager_function(
        &star_path,
        &entry.function_name,
        manager_name,
        package_name,
        config.state.clone(),
        None,
    ) {
        Ok(mut versions) => {
            info!("found {} versions", versions.len());
            versions.sort_by(|a, b| {
                b.release_date
                    .cmp(&a.release_date)
                    .then_with(|| b.version.cmp(&a.version))
            });

            print_versions_table(&versions);

            if let Some(v) = versions.first() {
                info!("testing pipeline for version {}", v.version);
            }
        }
        Err(e) => error!("mgr function failed: {}", e),
    }
}

fn run_package_function(config: &Config, package_name: &str, entry: &crate::models::package_entry::PackageEntry) {
    info!(
        "matched pkg: {} calling {} from {}",
        package_name, entry.function_name, entry.filename
    );

    let star_path = Path::new(&entry.filename);
    match crate::starlark::runtime::execute_function(
        &star_path,
        &entry.function_name,
        package_name,
        config.state.clone(),
        None,
    ) {
        Ok(mut versions) => {
            info!("found {} versions", versions.len());
            versions.sort_by(|a, b| {
                b.release_date
                    .cmp(&a.release_date)
                    .then_with(|| b.version.cmp(&a.version))
            });

            print_versions_table(&versions);

            if let Some(v) = versions.first() {
                info!("testing pipeline for version {}", v.version);
            }
        }
        Err(e) => error!("function failed: {}", e),
    }
}

fn print_versions_table(versions: &[VersionEntry]) {
    if versions.is_empty() {
        return;
    }

    let mut table = comfy_table::Table::new();
    table.load_preset(comfy_table::presets::NOTHING);
    table.set_header(vec![
        "Package",
        "Version",
        "Stream",
        "Release Date",
        "Type",
        "Steps",
    ]);

    for v in versions.iter().take(5) {
        let stream = if v.stream.is_empty() { "-" } else { &v.stream };
        table.add_row(vec![
            &v.pkgname,
            &v.version,
            stream,
            &v.release_date,
            &v.release_type,
            &v.pipeline.len().to_string(),
        ]);
    }

    println!("{}", table);
}
