use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::models::version_entry::VersionList;
use comfy_table::presets::NOTHING;
use comfy_table::Table;

pub fn run(config: &Config, selector_str: Option<&str>) {
    let selector = selector_str.and_then(PackageSelector::parse);

    let repo_config = Repositories::get_all(config);

    let mut table = Table::new();
    table.load_preset(NOTHING);
    table.set_header(vec!["Repo", "Package", "Version", "Date", "Type"]);

    let target_version = selector
        .as_ref()
        .and_then(|s| s.version.clone())
        .unwrap_or_else(|| "stable".to_string());

    for repo in &repo_config.repositories {
        if let Some(ref s) = selector {
            if let Some(ref r_name) = s.recipe {
                if repo.name != *r_name {
                    continue;
                }
            }
        }

        if let Some(pkg_list) = PackageList::get_for_repo(config, repo) {
            // If no selector, try to show the latest version from cache for each package
            if selector.is_none() {
                for pkg in &pkg_list.packages {
                    if let Ok(v_list) = VersionList::load(config, &repo.name, &pkg.name) {
                        add_versions_to_table(&mut table, repo.name.clone(), v_list, "latest", 1);
                    } else {
                        table.add_row(vec![
                            repo.name.clone(),
                            pkg.name.clone(),
                            "-".to_string(),
                            "-".to_string(),
                            "-".to_string(),
                        ]);
                    }
                }
                continue;
            }

            let s = selector.as_ref().unwrap();

            // Filter packages if a package name is provided
            if !s.package.is_empty() && s.prefix.is_none() {
                for pkg in &pkg_list.packages {
                    if s.package != "*" && pkg.name != s.package {
                        continue;
                    }

                    if let Some(v_list) = VersionList::get_for_package(config, repo, &pkg.name, Some(pkg), None) {
                        add_versions_to_table(
                            &mut table,
                            repo.name.clone(),
                            (*v_list).clone(),
                            &target_version,
                            5,
                        );
                    }
                }
            }

            // Handle managers if prefix is present
            if let Some(ref prefix) = s.prefix {
                if let Some(mgr) = pkg_list.manager_map.get(prefix) {
                    if s.package.is_empty() {
                        // Just list the manager itself
                        table.add_row(vec![
                            repo.name.clone(),
                            format!("{}:*", prefix),
                            "-".to_string(),
                            "-".to_string(),
                            "manager".to_string(),
                        ]);
                    } else {
                        let full_name = format!("{}:{}", prefix, s.package);
                        if let Some(v_list) =
                            VersionList::get_for_package(config, repo, &full_name, None, Some((mgr, &s.package)))
                        {
                            add_versions_to_table(
                                &mut table,
                                repo.name.clone(),
                                (*v_list).clone(),
                                &target_version,
                                5,
                            );
                        }
                    }
                }
            }
        }
    }

    println!("{table}");
}

fn add_versions_to_table(
    table: &mut Table,
    repo_name: String,
    v_list: VersionList,
    target_version: &str,
    limit: usize,
) {
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

    filtered_versions.sort_by(|a, b| b.release_date.cmp(&a.release_date).then_with(|| b.version.cmp(&a.version)));

    for v in filtered_versions.into_iter().take(limit) {
        table.add_row(vec![
            repo_name.clone(),
            v.pkgname,
            v.version,
            v.release_date,
            v.release_type,
        ]);
    }
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
