use crate::models::config::Config;
use crate::models::package_entry::{PackageList, PackageEntry};
use crate::models::repository::{Repositories, Repository};
use crate::models::selector::PackageSelector;
use crate::models::version_entry::VersionList;
use crate::utils::version::match_version_with_wildcard;
use comfy_table::presets::NOTHING;
use comfy_table::Table;
use std::sync::Arc;

/// Context for listing packages.
struct ListContext<'a> {
    config: &'a Config,
    selector: Option<PackageSelector>,
    all: bool,
    target_version: String,
    truncate: bool,
}

pub fn run(config: &Config, selector_str: Option<&str>, all: bool) {
    let selector = selector_str.and_then(PackageSelector::parse);
    let repo_config = Repositories::get_all(config);

    let (target_version, truncate) = determine_listing_mode(all, &selector);

    let ctx = ListContext {
        config,
        selector,
        all,
        target_version,
        truncate,
    };

    let mut table = create_list_table();

    for repo in &repo_config.repositories {
        if should_skip_repo(repo, &ctx.selector) {
            continue;
        }

        if let Some(pkg_list) = PackageList::get_for_repo(config, repo, false) {
            process_repo_packages(&ctx, repo, &pkg_list, &mut table);
        }
    }

    println!("{table}");
}

fn determine_listing_mode(all: bool, selector: &Option<PackageSelector>) -> (String, bool) {
    if all {
        ("all".to_string(), false)
    } else if selector.is_none() {
        ("stable".to_string(), true)
    } else {
        (
            selector
                .as_ref()
                .and_then(|s| s.version.clone())
                .unwrap_or_else(|| "stable".to_string()),
            false,
        )
    }
}

fn create_list_table() -> Table {
    let mut table = Table::new();
    table.load_preset(NOTHING);
    table.set_header(vec!["Repo", "Package", "Version", "Stream", "Date", "Type"]);
    table
}

fn should_skip_repo(repo: &Repository, selector: &Option<PackageSelector>) -> bool {
    if let Some(s) = selector {
        if let Some(r_name) = &s.recipe {
            return repo.name != *r_name;
        }
    }
    false
}

fn process_repo_packages(
    ctx: &ListContext,
    repo: &Repository,
    pkg_list: &PackageList,
    table: &mut Table,
) {
    if ctx.selector.is_none() {
        list_cached_packages(ctx, repo, pkg_list, table);
    } else {
        list_filtered_packages(ctx, repo, pkg_list, table);
    }
}

fn list_cached_packages(ctx: &ListContext, repo: &Repository, pkg_list: &PackageList, table: &mut Table) {
    for pkg in pkg_list.packages.values() {
        if let Ok(v_list) = VersionList::load(ctx.config, &repo.name, &pkg.name) {
            add_versions_to_table(table, &repo.name, v_list, &ctx.target_version, ctx.truncate);
        } else if !ctx.all {
            table.add_row(vec![
                repo.name.clone(),
                pkg.name.clone(),
                "-".to_string(),
                "-".to_string(),
                "-".to_string(),
                "-".to_string(),
            ]);
        }
    }
}

fn list_filtered_packages(ctx: &ListContext, repo: &Repository, pkg_list: &PackageList, table: &mut Table) {
    let s = ctx.selector.as_ref().unwrap();

    // Filter packages if a package name is provided
    if !s.package.is_empty() && s.prefix.is_none() {
        for pkg in pkg_list.packages.values() {
            if s.package != "*" && pkg.name != s.package {
                continue;
            }

            if let Some(v_list) = VersionList::get_for_package(crate::models::version_entry::GetVersionOptions {
                config: ctx.config,
                repo,
                package_name: &pkg.name,
                package_entry: Some(pkg),
                manager_entry: None,
                force: false,
            }) {
                add_versions_to_table(table, &repo.name, (*v_list).clone(), &ctx.target_version, ctx.truncate);
            }
        }
    }

    // Handle managers if prefix is present
    if let Some(prefix) = &s.prefix {
        handle_manager_listing(ctx, repo, pkg_list, prefix, table);
    }
}

fn handle_manager_listing(
    ctx: &ListContext,
    repo: &Repository,
    pkg_list: &PackageList,
    prefix: &str,
    table: &mut Table,
) {
    if let Some(mgr) = pkg_list.managers.get(prefix) {
        let s = ctx.selector.as_ref().unwrap();
        if s.package.is_empty() {
            table.add_row(vec![
                repo.name.clone(),
                format!("{}:*", prefix),
                "-".to_string(),
                "-".to_string(),
                "-".to_string(),
                "manager".to_string(),
            ]);
        } else {
            let full_name = format!("{}:{}", prefix, &s.package);
            if let Some(v_list) = VersionList::get_for_package(crate::models::version_entry::GetVersionOptions {
                config: ctx.config,
                repo,
                package_name: &full_name,
                package_entry: None,
                manager_entry: Some((mgr, &s.package)),
                force: false,
            }) {
                add_versions_to_table(table, &repo.name, (*v_list).clone(), &ctx.target_version, ctx.truncate);
            }
        }
    }
}

fn add_versions_to_table(
    table: &mut Table,
    repo_name: &str,
    v_list: VersionList,
    target_version: &str,
    truncate: bool,
) {
    let mut filtered_versions: Vec<_> = v_list.versions.into_iter().filter(|v| match_version(v, target_version)).collect();

    filtered_versions.sort_by(|a, b| {
        b.version.cmp(&a.version).then_with(|| b.release_date.cmp(&a.release_date))
    });

    if truncate && !filtered_versions.is_empty() {
        filtered_versions.truncate(1);
    }

    for v in filtered_versions {
        table.add_row(vec![
            repo_name.to_string(),
            v.pkgname,
            v.version.to_string(),
            if v.stream.is_empty() { "-".to_string() } else { v.stream },
            v.release_date,
            v.release_type.to_string(),
        ]);
    }
}

fn match_version(v: &crate::models::version_entry::VersionEntry, target: &str) -> bool {
    match target {
        "all" => true,
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
