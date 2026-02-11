use crate::models::repository::{Repository, RepositoryConfig};
use crate::models::package_entry::{PackageEntry, PackageList};
use crate::models::version_entry::VersionList;
use crate::models::selector::PackageSelector;
use crate::starlark::runtime::execute_function;
use crate::commands::package::list;
use std::fs;
use std::path::Path;

pub fn run(selector_str: Option<&str>) {
    let selector = selector_str.and_then(PackageSelector::parse);
    
    let config_dir = dirs_next::config_dir()
        .expect("Failed to get config directory")
        .join("pi");
    let config_file = config_dir.join("repositories.json");

    if !config_file.exists() {
        println!("No repositories configured.");
        return;
    }

    let content = fs::read_to_string(&config_file).expect("Failed to read config file");
    let config: RepositoryConfig = serde_json::from_str(&content).expect("Failed to parse config file");

    let cache_dir = dirs_next::cache_dir()
        .expect("Failed to get cache directory")
        .join("pi")
        .join("meta");
    fs::create_dir_all(&cache_dir).expect("Failed to create cache directory");
    let download_dir = dirs_next::cache_dir()
        .expect("Failed to get cache directory")
        .join("pi")
        .join("downloads");

    for repo in &config.repositories {
        // If recipe is specified, it must match repo name
        if let Some(ref s) = selector {
            if let Some(ref r_name) = s.recipe {
                if repo.name != *r_name {
                    continue;
                }
            }
        }

        let repo_cache_file = cache_dir.join(format!("packages-{}.json", repo.uuid));
        if !repo_cache_file.exists() {
            continue;
        }

        let pkg_content = fs::read_to_string(&repo_cache_file).expect("Failed to read repo cache file");
        let pkg_list: PackageList = serde_json::from_str(&pkg_content).expect("Failed to parse repo cache file");

        for pkg in pkg_list.packages {
            // Match package name
            if let Some(ref s) = selector {
                if !s.package.is_empty() && s.package != "*" {
                    // Check if selector.package is a substring of pkg.name or matches exactly
                    // This allows "node" to match "^node"
                    if !pkg.name.contains(&s.package) {
                        continue;
                    }
                }
            }

            // Prefix handling:
            // "prefixed packages are not synced unless full name is provideded"
            // If the package name (name) contains a colon, it's prefixed.
            if pkg.name.contains(':') {
                if let Some(ref s) = selector {
                    if s.prefix.is_none() {
                        // Prefixed but no prefix in selector (unless selector is specific)
                        // Actually the requirement says "unless full name is provided"
                        // I'll assume if selector.package matches pkg.name exactly it's fine.
                        if pkg.name != s.package {
                            continue;
                        }
                    }
                } else {
                    // No selector, skip prefixed packages
                    continue;
                }
            }

            sync_package(repo, &pkg, &cache_dir, &download_dir);
        }
    }

    list::run(selector_str);
}

fn sync_package(repo: &Repository, pkg: &PackageEntry, cache_dir: &Path, download_dir: &Path) {
    println!("Syncing package: {} in repo: {}...", pkg.name, repo.name);
    
    let star_path = Path::new(&repo.path).join(&pkg.filename);
    match execute_function(&star_path, &pkg.function_name, &pkg.name, download_dir.to_path_buf()) {
        Ok(versions) => {
            let version_list = VersionList { versions };
            let version_cache_file = cache_dir.join(format!("version-{}-{}.json", repo.uuid, pkg.name));
            let content = serde_json::to_string_pretty(&version_list).expect("Failed to serialize version list");
            fs::write(&version_cache_file, content).expect("Failed to write version cache file");
            println!("Synced {} versions for {}", version_list.versions.len(), pkg.name);
        }
        Err(e) => {
            eprintln!("Error syncing package {}: {}", pkg.name, e);
        }
    }
}
