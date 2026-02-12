use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::models::version_entry::{VersionEntry, VersionList};
use comfy_table::Table;

pub fn run(config: &Config, queries: Vec<String>) {
    let repo_config = Repositories::get_all(config);

    // Resolve queries in parallel
    use rayon::prelude::*;
    let results: Vec<(String, String, String)> = queries
        .par_iter()
        .map(|query| {
            let selector = match PackageSelector::parse(query) {
                Some(s) => s,
                None => {
                    return (
                        query.clone(),
                        "Invalid selector".to_string(),
                        "-".to_string(),
                    )
                }
            };

            match resolve_query(config, repo_config, &selector) {
                Some((full_qualified_name, version)) => {
                    (query.clone(), full_qualified_name, version.release_date)
                }
                None => (query.clone(), "Not found".to_string(), "-".to_string()),
            }
        })
        .collect();

    // Print results
    let mut table = Table::new();
    table.set_header(vec!["Query", "Resolved Full Name", "Release Date"]);
    for (query, full_name, date) in results {
        table.add_row(vec![query, full_name, date]);
    }
    println!("{table}");
}

pub fn resolve_query(
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

        let pkg_list = match PackageList::get_for_repo(config, repo) {
            Some(l) => l,
            None => continue,
        };

        // Check direct packages
        if selector.prefix.is_none() {
            if let Some(pkg) = pkg_list.package_map.get(&selector.package) {
                if let Some(v_list) =
                    VersionList::get_for_package(config, repo, &pkg.name, Some(pkg), None)
                {
                    if let Some(v) = find_best_version((*v_list).clone(), target_version) {
                        let full_qualified = format!("{}/{}={}", repo.name, pkg.name, v.version);
                        return Some((full_qualified, v));
                    }
                }
            }
        }

        // Check managers
        if let Some(ref prefix) = selector.prefix {
            if let Some(mgr) = pkg_list.manager_map.get(prefix) {
                let full_name = format!("{}:{}", prefix, selector.package);
                if let Some(v_list) = VersionList::get_for_package(
                    config,
                    repo,
                    &full_name,
                    None,
                    Some((mgr, &selector.package)),
                ) {
                    if let Some(v) = find_best_version((*v_list).clone(), target_version) {
                        let full_qualified =
                            format!("{}/{}={}", repo.name, full_name, v.version);
                        return Some((full_qualified, v));
                    }
                }
            }
        }
    }
    None
}

pub fn find_best_version(v_list: VersionList, target_version: &str) -> Option<VersionEntry> {
    let mut filtered_versions: Vec<_> = v_list
        .versions
        .into_iter()
        .filter(|v| match target_version {
            "latest" => true,
            "stable" | "lts" | "testing" | "unstable" => {
                v.release_type.to_lowercase() == target_version
            }
            _ => {
                if target_version.contains('*') {
                    match_version_with_wildcard(&v.version, target_version)
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

fn match_version_with_wildcard(version: &str, pattern: &str) -> bool {
    let version_parts: Vec<&str> = version.split('.').collect();
    let pattern_parts: Vec<&str> = pattern.split('.').collect();

    if version_parts.len() != pattern_parts.len() {
        return false;
    }

    for (v, p) in version_parts.iter().zip(pattern_parts.iter()) {
        if *p != "*" && v != p {
            return false;
        }
    }
    true
}
