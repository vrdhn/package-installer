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
            for pkg in &pkg_list.packages {
                if let Some(ref s) = selector {
                    if !s.package.is_empty() && s.package != "*" {
                        if pkg.name != s.package {
                            continue;
                        }
                    }
                }

                if let Some(v_list) = VersionList::get_for_package(config, repo, &pkg.name, Some(pkg), None) {
                    add_versions_to_table(
                        &mut table,
                        repo.name.clone(),
                        (*v_list).clone(),
                        &target_version,
                    );
                }
            }

            // Handle managers
            if let Some(ref s) = selector {
                if let Some(ref prefix) = s.prefix {
                    if let Some(mgr) = pkg_list.manager_map.get(prefix) {
                        let full_name = format!("{}:{}", prefix, s.package);
                        if let Some(v_list) =
                            VersionList::get_for_package(config, repo, &full_name, None, Some((mgr, &s.package)))
                        {
                            add_versions_to_table(
                                &mut table,
                                repo.name.clone(),
                                (*v_list).clone(),
                                &target_version,
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

    // Sort by release_date descending
    filtered_versions.sort_by(|a, b| b.release_date.cmp(&a.release_date));

    for v in filtered_versions.into_iter().take(5) {
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
