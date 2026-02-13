use crate::models::config::Config;
use crate::models::package_entry::PackageEntry;
use crate::models::version_entry::VersionEntry;
use crate::starlark::runtime::{evaluate_file, execute_function};
use crate::services::downloader::Downloader;
use crate::services::unarchiver::Unarchiver;
use comfy_table::presets::NOTHING;
use comfy_table::Table;
use log::{error, info};
use std::path::Path;

pub fn run(config: &Config, filename: &str, pkg: Option<&str>) {
    info!("testing file: {}", filename);

    let path = Path::new(filename);
    match evaluate_file(path, config.state.clone()) {
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
    ) {
        Ok(mut versions) => {
            info!("found {} versions", versions.len());

            // Sort by date then by version
            versions.sort_by(|a, b| {
                b.release_date
                    .cmp(&a.release_date)
                    .then_with(|| b.version.cmp(&a.version))
            });

            print_versions_table(&versions);

            // For testing, just take the first one (latest)
            if let Some(v) = versions.first() {
                if let Err(e) = test_package_download_unarchive(config, v, "devel-test") {
                    error!("download/unarchive failed: {}", e);
                }
            }
        }
        Err(e) => error!("mgr function failed: {}", e),
    }
}

fn run_package_function(config: &Config, package_name: &str, entry: &PackageEntry) {
    info!(
        "matched pkg: {} calling {} from {}",
        package_name, entry.function_name, entry.filename
    );

    let star_path = Path::new(&entry.filename);
    match execute_function(
        &star_path,
        &entry.function_name,
        package_name,
        config.state.clone(),
    ) {
        Ok(mut versions) => {
            info!("found {} versions", versions.len());

            // Sort by date then by version
            versions.sort_by(|a, b| {
                b.release_date
                    .cmp(&a.release_date)
                    .then_with(|| b.version.cmp(&a.version))
            });

            print_versions_table(&versions);

            // For testing, just take the first one (latest)
            if let Some(v) = versions.first() {
                if let Err(e) = test_package_download_unarchive(config, v, "devel-test") {
                    error!("download/unarchive failed: {}", e);
                }
            }
        }
        Err(e) => error!("function failed: {}", e),
    }
}

fn test_package_download_unarchive(config: &Config, v: &VersionEntry, repo_name: &str) -> anyhow::Result<()> {
    info!("testing download & unarchive");
    
    let download_dest = config.download_dir.join(&v.filename);
    let checksum = if v.checksum.is_empty() { None } else { Some(v.checksum.as_str()) };

    Downloader::download_to_file(&v.url, &download_dest, checksum)?;

    let pkg_dir_name = format!("{}-{}-{}", sanitize_name(&v.pkgname), sanitize_name(&v.version), repo_name);
    let extract_dest = config.packages_dir.join(pkg_dir_name);

    Unarchiver::unarchive(&download_dest, &extract_dest)?;

    info!("test success");
    Ok(())
}

fn sanitize_name(name: &str) -> String {
    name.replace(['/', '\\', ' ', ':'], "_")
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

    for v in versions.iter().take(5) {
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
