use crate::models::config::Config;
use crate::models::package_entry::{ManagerEntry, PackageEntry, PackageList};
use crate::models::repository::Repository;
use crate::models::version_entry::VersionList;
use crate::starlark::runtime::{evaluate_file, execute_function, execute_manager_function};
use std::path::Path;
use walkdir::WalkDir;

pub fn sync_repo(config: &Config, repo: &Repository) {
    println!("Syncing repository: {}...", repo.name);
    let mut all_packages = Vec::new();
    let mut all_managers = Vec::new();
    let repo_path = Path::new(&repo.path);

    for entry in WalkDir::new(repo_path)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| e.path().extension().map_or(false, |ext| ext == "star"))
    {
        let star_file_path = entry.path();
        match evaluate_file(star_file_path, config.download_dir.clone(), config.state.clone()) {
            Ok((packages, managers)) => {
                let rel_path = star_file_path
                    .strip_prefix(repo_path)
                    .unwrap_or(star_file_path)
                    .to_string_lossy()
                    .to_string();

                for mut pkg in packages {
                    pkg.filename = rel_path.clone();
                    all_packages.push(pkg);
                }
                for mut mgr in managers {
                    mgr.filename = rel_path.clone();
                    all_managers.push(mgr);
                }
            }
            Err(e) => {
                eprintln!("Error evaluating {}: {}", star_file_path.display(), e);
            }
        }
    }

    let mut package_list = PackageList {
        packages: all_packages,
        managers: all_managers,
        package_map: std::collections::HashMap::new(),
        manager_map: std::collections::HashMap::new(),
    };
    package_list.initialize_maps();
    package_list
        .save(config, &repo.uuid)
        .expect("Failed to save package list");
    println!(
        "Synced {} packages and {} managers for {}",
        package_list.packages.len(),
        package_list.managers.len(),
        repo.name
    );
}

pub fn sync_package(config: &Config, repo: &Repository, pkg: &PackageEntry) {
    println!("Syncing package: {} in repo: {}...", pkg.name, repo.name);

    let star_path = Path::new(&repo.path).join(&pkg.filename);
    match execute_function(
        &star_path,
        &pkg.function_name,
        &pkg.name,
        config.download_dir.clone(),
        config.state.clone(),
    ) {
        Ok(versions) => {
            let version_list = VersionList { versions };
            version_list
                .save(config, &repo.uuid, &pkg.name)
                .expect("Failed to save version list");
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

pub fn sync_manager_package(
    config: &Config,
    repo: &Repository,
    mgr: &ManagerEntry,
    manager_name: &str,
    package_name: &str,
) {
    println!(
        "Syncing package: {}:{} using manager: {} in repo: {}...",
        manager_name, package_name, mgr.name, repo.name
    );

    let star_path = Path::new(&repo.path).join(&mgr.filename);
    match execute_manager_function(
        &star_path,
        &mgr.function_name,
        manager_name,
        package_name,
        config.download_dir.clone(),
        config.state.clone(),
    ) {
        Ok(versions) => {
            let version_list = VersionList { versions };
            let full_name = format!("{}:{}", manager_name, package_name);
            version_list
                .save(config, &repo.uuid, &full_name)
                .expect("Failed to save version list");
            println!(
                "Synced {} versions for {}",
                version_list.versions.len(),
                full_name
            );
        }
        Err(e) => {
            eprintln!(
                "Error syncing package {}:{}: {}",
                manager_name, package_name, e
            );
        }
    }
}
