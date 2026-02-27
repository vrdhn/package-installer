use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::{Repositories, Repository};
use crate::models::selector::PackageSelector;
use crate::models::version_entry::{VersionEntry, VersionList};
use crate::utils::version::match_version_with_wildcard;
use comfy_table::presets::NOTHING;
use comfy_table::Table;
use rayon::prelude::*;

/// Runs the package resolution for multiple queries in parallel.
pub fn run(config: &Config, queries: Vec<String>) {
    let repo_config = Repositories::get_all(config);

    let results: Vec<(String, String, String)> = queries
        .par_iter()
        .map(|query| resolve_single_query(config, &repo_config, query))
        .collect();

    print_resolution_table(results);
}

fn resolve_single_query(config: &Config, repo_config: &Repositories, query: &str) -> (String, String, String) {
    let selector = match PackageSelector::parse(query) {
        Some(s) => s,
        None => return (query.to_string(), "Invalid selector".to_string(), "-".to_string()),
    };

    match resolve_query(config, repo_config, &selector) {
        Some((full_qualified_name, version, _repo_name)) => {
            (query.to_string(), full_qualified_name, version.release_date)
        }
        None => (query.to_string(), "Not found".to_string(), "-".to_string()),
    }
}

fn print_resolution_table(results: Vec<(String, String, String)>) {
    let mut table = Table::new();
    table.load_preset(NOTHING);
    table.set_header(vec!["Query", "Resolved Full Name", "Release Date"]);
    for (query, full_name, date) in results {
        table.add_row(vec![query, full_name, date]);
    }
    println!("{table}");
}

/// Resolves a single query against available repositories.
/// Example selector: "pi:rust/cargo=1.70.0"
pub fn resolve_query(
    config: &Config,
    repo_config: &Repositories,
    selector: &PackageSelector,
) -> Option<(String, VersionEntry, String)> {
    // Try cached first
    if let Some(res) = resolve_query_internal(config, repo_config, selector, false) {
        return Some(res);
    }

    // Attempt sync if allowed
    if !config.force {
        log::debug!("[{}] not found in cache, attempting sync", selector.package);
        return resolve_query_internal(config, repo_config, selector, true);
    }

    None
}

fn resolve_query_internal(
    config: &Config,
    repo_config: &Repositories,
    selector: &PackageSelector,
    force: bool,
) -> Option<(String, VersionEntry, String)> {
    let target_version = selector.version.as_deref().unwrap_or("stable");

    for repo in &repo_config.repositories {
        if should_skip_repo(repo, selector) { continue; }

        let pkg_list = PackageList::get_for_repo(config, repo, force)?;
        
        if let Some(res) = try_resolve_in_repo(config, repo, &pkg_list, selector, target_version, force) {
            return Some(res);
        }
    }
    None
}

fn should_skip_repo(repo: &Repository, selector: &PackageSelector) -> bool {
    selector.recipe.as_ref().map_or(false, |r| repo.name != *r)
}

struct ResolveOptions<'a> {
    config: &'a Config,
    repo: &'a Repository,
    package_name: &'a str,
    pkg_entry: Option<&'a crate::models::package_entry::PackageEntry>,
    mgr_entry: Option<(&'a crate::models::package_entry::ManagerEntry, &'a str)>,
    target_version: &'a str,
    force: bool,
}

fn try_resolve_in_repo(
    config: &Config,
    repo: &Repository,
    pkg_list: &PackageList,
    selector: &PackageSelector,
    target_version: &str,
    force: bool,
) -> Option<(String, VersionEntry, String)> {
    // 1. Direct package resolution
    if selector.prefix.is_none() {
        if let Some(pkg) = pkg_list.packages.get(&selector.package) {
            let res = resolve_version(ResolveOptions {
                config, repo, package_name: &pkg.name, pkg_entry: Some(pkg),
                mgr_entry: None, target_version, force,
            });
            if let Some(v) = res {
                let full_qualified = format!("{}/{}={}", repo.name, pkg.name, v.version);
                return Some((full_qualified, v, repo.name.clone()));
            }
        }
    }

    // 2. Manager-based resolution
    if let Some(ref prefix) = selector.prefix {
        if let Some(mgr) = pkg_list.managers.get(prefix) {
            let full_name = format!("{}:{}", prefix, selector.package);
            let res = resolve_version(ResolveOptions {
                config, repo, package_name: &full_name, pkg_entry: None,
                mgr_entry: Some((mgr, &selector.package)), target_version, force,
            });
            if let Some(v) = res {
                let full_qualified = format!("{}/{}={}", repo.name, full_name, v.version);
                return Some((full_qualified, v, repo.name.clone()));
            }
        }
    }
    None
}

fn resolve_version(opts: ResolveOptions) -> Option<VersionEntry> {
    let v_list = VersionList::get_for_package(crate::models::version_entry::GetVersionOptions {
        config: opts.config,
        repo: opts.repo,
        package_name: opts.package_name,
        package_entry: opts.pkg_entry,
        manager_entry: opts.mgr_entry,
        force: opts.force,
    })?;
    find_best_version((*v_list).clone(), opts.target_version)
}

pub fn find_best_version(v_list: VersionList, target_version: &str) -> Option<VersionEntry> {
    let mut filtered_versions: Vec<_> = v_list.versions.into_iter().filter(|v| match_target_version(v, target_version)).collect();

    filtered_versions.sort_by(|a, b| {
        b.version.cmp(&a.version).then_with(|| b.release_date.cmp(&a.release_date))
    });

    filtered_versions.into_iter().next()
}

fn match_target_version(v: &VersionEntry, target: &str) -> bool {
    match target {
        "latest" => true,
        "stable" | "lts" | "testing" | "unstable" => v.release_type.to_string().to_lowercase() == target,
        _ => {
            if target.contains('*') {
                match_version_with_wildcard(&v.version.to_string(), target)
            } else {
                v.version.to_string() == target
            }
        }
    }
}
