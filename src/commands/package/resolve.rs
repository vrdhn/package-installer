use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::models::version_entry::{VersionEntry, VersionList};
use comfy_table::Table;
use glob::Pattern;
use std::fs;

pub fn run(config: &Config, queries: Vec<String>) {
    let config_file = config.repositories_file();

    if !config_file.exists() {
        println!("No repositories configured.");
        return;
    }

    let content = fs::read_to_string(&config_file).expect("Failed to read config file");
    let repo_config: Repositories =
        serde_json::from_str(&content).expect("Failed to parse config file");

    let mut table = Table::new();
    table.set_header(vec!["Query", "Resolved Full Name", "Release Date"]);

    for query in queries {
        let selector = match PackageSelector::parse(&query) {
            Some(s) => s,
            None => {
                table.add_row(vec![
                    query,
                    "Invalid selector".to_string(),
                    "-".to_string(),
                ]);
                continue;
            }
        };

        let resolved = resolve_query(config, &repo_config, &selector);
        match resolved {
            Some((full_qualified_name, version)) => {
                table.add_row(vec![
                    query,
                    full_qualified_name,
                    version.release_date,
                ]);
            }
            None => {
                table.add_row(vec![
                    query,
                    "Not found".to_string(),
                    "-".to_string(),
                ]);
            }
        }
    }

    println!("{table}");
}

fn resolve_query(
    config: &Config,
    repo_config: &Repositories,
    selector: &PackageSelector,
) -> Option<(String, VersionEntry)> {
    let target_version = selector.version.as_deref().unwrap_or("stable");

    for repo in &repo_config.repositories {
        if let Some(ref r_name) = selector.recipe {
            if repo.name != *r_name {
                continue;
            }
        }

        let repo_cache_file = config.package_cache_file(&repo.uuid);
        if !repo_cache_file.exists() {
            continue;
        }

        let pkg_content =
            fs::read_to_string(&repo_cache_file).expect("Failed to read repo cache file");
        let pkg_list: PackageList =
            serde_json::from_str(&pkg_content).expect("Failed to parse repo cache file");

        // Check direct packages
        if selector.prefix.is_none() {
            for pkg in &pkg_list.packages {
                if pkg.name == selector.package || pkg.name.contains(&selector.package) {
                    let safe_name = pkg.name.replace('/', "#");
                    let version_cache_file = config.version_cache_file(&repo.uuid, &safe_name);
                    if let Some(v) = find_best_version(&version_cache_file, target_version) {
                        let full_qualified = format!("{}/{}={}", repo.name, pkg.name, v.version);
                        return Some((full_qualified, v));
                    }
                }
            }
        }

        // Check installers
        if let Some(ref prefix) = selector.prefix {
            for inst in &pkg_list.installers {
                if inst.name == *prefix {
                    let full_name = format!("{}:{}", prefix, selector.package);
                    let safe_name = full_name.replace('/', "#");
                    let version_cache_file = config.version_cache_file(&repo.uuid, &safe_name);
                    if let Some(v) = find_best_version(&version_cache_file, target_version) {
                        let full_qualified = format!("{}/{}={}", repo.name, full_name, v.version);
                        return Some((full_qualified, v));
                    }
                }
            }
        }
    }

    None
}

fn find_best_version(cache_file: &std::path::Path, target_version: &str) -> Option<VersionEntry> {
    if !cache_file.exists() {
        return None;
    }

    let v_content = fs::read_to_string(cache_file).ok()?;
    let v_list: VersionList = serde_json::from_str(&v_content).ok()?;

    let mut filtered_versions: Vec<_> = v_list
        .versions
        .into_iter()
        .filter(|v| match target_version {
            "latest" => true,
            "stable" | "lts" | "testing" | "unstable" => {
                v.release_type.to_lowercase() == target_version
            }
            _ => {
                if target_version.contains('*') || target_version.contains('?') {
                    if let Ok(pattern) = Pattern::new(target_version) {
                        pattern.matches(&v.version)
                    } else {
                        v.version == target_version
                    }
                } else {
                    v.version == target_version
                }
            }
        })
        .collect();

    // Sort by release_date descending
    filtered_versions.sort_by(|a, b| b.release_date.cmp(&a.release_date));

    filtered_versions.into_iter().next()
}
