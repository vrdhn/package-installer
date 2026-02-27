use anyhow::{Context, Result};
use log::{error, info};
use crate::models::config::Config;
use crate::models::package_entry::{PackageEntry, ManagerEntry, PackageList, RegistryEntry};
use crate::models::repository::Repository;
use crate::models::version_entry::{VersionEntry, VersionList};
use crate::starlark::runtime::{evaluate_file, execute_function, execute_manager_function, ExecutionOptions};
use std::path::Path;
use std::collections::HashMap;
use walkdir::WalkDir;

/// Synchronizes a repository by evaluating all `.star` files and saving the package list.
pub fn sync_repo(config: &Config, repo: &Repository) -> Result<()> {
    info!("[{}] syncing repo", repo.name);
    
    // Clear old cache files and in-memory entries for this repo to ensure a clean slate.
    clear_repo_cache(config, &repo.name)?;

    let (packages, managers) = collect_repo_entries(config, repo);

    let package_list = PackageList {
        packages,
        managers,
    };
    package_list
        .save(config, &repo.name)
        .context("Failed to save package list")?;

    info!(
        "[{}] synced: {} pkgs, {} mgrs",
        repo.name,
        package_list.packages.len(),
        package_list.managers.len()
    );
    Ok(())
}

fn clear_repo_cache(config: &Config, repo_name: &str) -> Result<()> {
    // 1. Clear in-memory caches
    config.state.package_lists.remove(repo_name);
    config.state.version_lists.retain(|k, _| !k.starts_with(&format!("{}:", repo_name)));

    // 2. Clear version cache files from disk
    let prefix = format!("version-{}-", repo_name);
    if let Ok(entries) = std::fs::read_dir(&config.cache_meta_dir) {
        for entry in entries.filter_map(|e| e.ok()) {
            if let Some(name) = entry.file_name().to_str() {
                if name.starts_with(&prefix) && name.ends_with(".json") {
                    let _ = std::fs::remove_file(entry.path());
                }
            }
        }
    }
    Ok(())
}

/// Iterates through the repository, evaluates Starlark files, and collects package/manager entries.
fn collect_repo_entries(config: &Config, repo: &Repository) -> (HashMap<String, RegistryEntry>, HashMap<String, RegistryEntry>) {
    let repo_path = Path::new(&repo.path);
    WalkDir::new(repo_path)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| e.path().extension().map_or(false, |ext| ext == "star"))
        .fold((HashMap::new(), HashMap::new()), |(mut pkgs, mut mgrs), entry| {
            let star_file_path = entry.path();
            match evaluate_file(star_file_path, config) {
                Ok((found_pkgs, found_mgrs)) => {
                    let rel_path = star_file_path
                        .strip_prefix(repo_path)
                        .unwrap_or(star_file_path)
                        .to_string_lossy()
                        .to_string();

                    for mut p in found_pkgs {
                        p.filename = rel_path.clone();
                        pkgs.insert(p.name.clone(), p);
                    }
                    for mut m in found_mgrs {
                        m.filename = rel_path.clone();
                        mgrs.insert(m.name.clone(), m);
                    }
                }
                Err(e) => {
                    error!("[{}] eval failed {}: {}", repo.name, star_file_path.display(), e);
                }
            }
            (pkgs, mgrs)
        })
}

/// Synchronizes a single package by executing its Starlark function and caching the versions.
pub fn sync_package(config: &Config, repo: &Repository, pkg: &PackageEntry) -> Result<()> {
    info!("{}/{} syncing pkg", repo.name, pkg.name);

    let star_path = Path::new(&repo.path).join(&pkg.filename);
    let versions = execute_function(
        ExecutionOptions {
            path: &star_path,
            function_name: &pkg.function_name,
            config,
            options: None,
        },
        &pkg.name,
    ).with_context(|| format!(
        "Failed to execute function '{}' in '{}' for package {}/{}", 
        pkg.function_name, star_path.display(), repo.name, pkg.name
    ))?;

    save_versions(config, &repo.name, &pkg.name, versions)
}

/// Synchronizes a package managed by a manager (e.g., go:pkg) by executing its manager function.
pub fn sync_manager_package(
    config: &Config,
    repo: &Repository,
    mgr: &ManagerEntry,
    manager_name: &str,
    package_name: &str,
) -> Result<()> {
    let full_name = format!("{}:{}", manager_name, package_name);
    info!("{}/{} syncing mgr pkg", repo.name, full_name);

    let star_path = Path::new(&repo.path).join(&mgr.filename);
    let versions = execute_manager_function(
        ExecutionOptions {
            path: &star_path,
            function_name: &mgr.function_name,
            config,
            options: None,
        },
        manager_name,
        package_name,
    ).with_context(|| format!(
        "Failed to execute manager function '{}' in '{}' for package {}/{}", 
        mgr.function_name, star_path.display(), repo.name, full_name
    ))?;

    save_versions(config, &repo.name, &full_name, versions)
}

/// Internal helper to save a list of versions to the cache.
fn save_versions(config: &Config, repo_name: &str, name: &str, versions: Vec<VersionEntry>) -> Result<()> {
    if versions.is_empty() {
        info!("{}/{} no versions found, not caching", repo_name, name);
        return Ok(());
    }

    let version_list = VersionList { versions };
    version_list
        .save(config, repo_name, name)
        .with_context(|| format!("Failed to save version list for package {}/{}", repo_name, name))?;

    info!(
        "{}/{} synced {} versions",
        repo_name,
        name,
        version_list.versions.len()
    );
    Ok(())
}
