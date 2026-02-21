use crate::models::config::Config;
use crate::models::version_entry::VersionEntry;
use log::{error, info};
use std::path::Path;

pub fn run(config: &Config, filename: &str, pkg: Option<&str>) {
    info!("testing file: {}", filename);

    let path = Path::new(filename);
    match crate::starlark::runtime::evaluate_file(path, config.state.clone()) {
        Ok((packages, _managers)) => {
            info!("registered {} pkgs", packages.len());
            if let Some(package_name) = pkg {
                if let Some(pkg_entry) = packages.iter().find(|p| p.name == package_name) {
                    run_package_function(config, package_name, pkg_entry);
                    return;
                }
                error!("pkg {} not found", package_name);
            }
        }
        Err(e) => error!("eval failed: {}", e),
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
        "Release Date",
        "Type",
        "Steps",
    ]);

    for v in versions.iter().take(5) {
        table.add_row(vec![
            &v.pkgname,
            &v.version,
            &v.release_date,
            &v.release_type,
            &v.pipeline.len().to_string(),
        ]);
    }

    println!("{}", table);
}
