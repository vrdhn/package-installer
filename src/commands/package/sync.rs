use crate::commands::package::list;
use crate::models::config::Config;
use crate::models::package_entry::{InstallerEntry, PackageEntry, PackageList};
use crate::models::repository::{Repository, Repositories};
use crate::models::selector::PackageSelector;
use crate::models::version_entry::VersionList;
use crate::starlark::runtime::{execute_function, execute_installer_function};
use rayon::prelude::*;
use std::fs;
use std::path::Path;

pub fn run(config: &Config, selector_str: Option<&str>) {
    let selector = selector_str.and_then(PackageSelector::parse);

    let config_file = config.repositories_file();

    if !config_file.exists() {
        println!("No repositories configured.");
        return;
    }

    let content = fs::read_to_string(&config_file).expect("Failed to read config file");
    let repo_config: Repositories =
        serde_json::from_str(&content).expect("Failed to parse config file");

    fs::create_dir_all(&config.meta_dir).expect("Failed to create cache directory");
    let download_dir = &config.download_dir;

    repo_config.repositories.par_iter().for_each(|repo| {
        // If recipe is specified, it must match repo name
        if let Some(ref s) = selector {
            if let Some(ref r_name) = s.recipe {
                if repo.name != *r_name {
                    return;
                }
            }
        }

        let repo_cache_file = config.package_cache_file(&repo.uuid);
        if !repo_cache_file.exists() {
            return;
        }

        let pkg_content =
            fs::read_to_string(&repo_cache_file).expect("Failed to read repo cache file");
        let pkg_list: PackageList =
            serde_json::from_str(&pkg_content).expect("Failed to parse repo cache file");

        pkg_list.packages.par_iter().for_each(|pkg| {
            // Match package name
            if let Some(ref s) = selector {
                if !s.package.is_empty() && s.package != "*" {
                    if !pkg.name.contains(&s.package) {
                        return;
                    }
                }
            }

            // Prefix handling:
            if pkg.name.contains(':') {
                if let Some(ref s) = selector {
                    if s.prefix.is_none() {
                        if pkg.name != s.package {
                            return;
                        }
                    }
                } else {
                    return;
                }
            }

            sync_package(config, repo, pkg, download_dir);
        });

        if let Some(ref s) = selector {
            if let Some(ref prefix) = s.prefix {
                pkg_list.installers.par_iter().for_each(|inst| {
                    if inst.name == *prefix {
                        sync_installer_package(config, repo, inst, prefix, &s.package, download_dir);
                    }
                });
            }
        }
    });

    list::run(config, selector_str);
}

fn sync_installer_package(
    config: &Config,
    repo: &Repository,
    inst: &InstallerEntry,
    installer_name: &str,
    package_name: &str,
    download_dir: &Path,
) {
    println!(
        "Syncing package: {}:{} using installer: {} in repo: {}...",
        installer_name, package_name, inst.name, repo.name
    );

    let star_path = Path::new(&repo.path).join(&inst.filename);
    match execute_installer_function(
        &star_path,
        &inst.function_name,
        installer_name,
        package_name,
        download_dir.to_path_buf(),
    ) {
        Ok(versions) => {
            let version_list = VersionList { versions };
            let full_name = format!("{}:{}", installer_name, package_name);
            let safe_name = full_name.replace('/', "#");
            let version_cache_file = config.version_cache_file(&repo.uuid, &safe_name);
            let content = serde_json::to_string_pretty(&version_list)
                .expect("Failed to serialize version list");
            fs::write(&version_cache_file, content).expect("Failed to write version cache file");
            println!(
                "Synced {} versions for {}",
                version_list.versions.len(),
                full_name
            );
        }
        Err(e) => {
            eprintln!(
                "Error syncing package {}:{}: {}",
                installer_name, package_name, e
            );
        }
    }
}

fn sync_package(config: &Config, repo: &Repository, pkg: &PackageEntry, download_dir: &Path) {
    println!("Syncing package: {} in repo: {}...", pkg.name, repo.name);

    let star_path = Path::new(&repo.path).join(&pkg.filename);
    match execute_function(
        &star_path,
        &pkg.function_name,
        &pkg.name,
        download_dir.to_path_buf(),
    ) {
        Ok(versions) => {
            let version_list = VersionList { versions };
            let safe_name = pkg.name.replace('/', "#");
            let version_cache_file = config.version_cache_file(&repo.uuid, &safe_name);
            let content = serde_json::to_string_pretty(&version_list)
                .expect("Failed to serialize version list");
            fs::write(&version_cache_file, content).expect("Failed to write version cache file");
            println!(
                "Synced {} versions for {}",
                version_list.versions.len(),
                pkg.name
            );
        }
        Err(e) => {
            eprintln!("Error syncing package {}: {}", pkg.name, e);
        }
    }
}
