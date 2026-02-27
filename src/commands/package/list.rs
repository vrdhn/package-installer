use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::models::version_entry::VersionList;
use crate::utils::version::match_version_with_wildcard;
use comfy_table::presets::NOTHING;
use comfy_table::Table;

pub fn run(config: &Config, selector_str: Option<&str>, all: bool) {
    let selector = selector_str.and_then(PackageSelector::parse);

    let repo_config = Repositories::get_all(config);

    let mut table = Table::new();
    table.load_preset(NOTHING);
    table.set_header(vec!["Repo", "Package", "Version", "Stream", "Date", "Type"]);

    let (target_version, truncate) = if all {
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
    };

    for repo in &repo_config.repositories {
        if let Some(ref s) = selector {
            if let Some(ref r_name) = s.recipe {
                if repo.name != *r_name {
                    continue;
                }
            }
        }

        if let Some(pkg_list) = PackageList::get_for_repo(config, repo, false) {
            // If no selector, try to show the versions from cache for each package
            if selector.is_none() {
                for pkg in &pkg_list.packages {
                    if let Ok(v_list) = VersionList::load(config, &repo.name, &pkg.name) {
                        add_versions_to_table(
                            &mut table,
                            repo.name.clone(),
                            v_list,
                            &target_version,
                            truncate,
                        );
                    } else {
                        if !all {
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
                continue;
            }

            let s = selector.as_ref().unwrap();

            // Filter packages if a package name is provided
            if !s.package.is_empty() && s.prefix.is_none() {
                for pkg in &pkg_list.packages {
                    if s.package != "*" && pkg.name != s.package {
                        continue;
                    }

                    if let Some(v_list) =
                        VersionList::get_for_package(config, repo, &pkg.name, Some(pkg), None, false)
                    {
                        add_versions_to_table(
                            &mut table,
                            repo.name.clone(),
                            (*v_list).clone(),
                            &target_version,
                            truncate,
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
                            "-".to_string(),
                            "manager".to_string(),
                        ]);
                    } else {
                        let full_name = format!("{}:{}", prefix, s.package);
                        if let Some(v_list) = VersionList::get_for_package(
                            config,
                            repo,
                            &full_name,
                            None,
                            Some((mgr, &s.package)),
                            false,
                        ) {
                            add_versions_to_table(
                                &mut table,
                                repo.name.clone(),
                                (*v_list).clone(),
                                &target_version,
                                truncate,
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
    truncate: bool,
) {
    let mut filtered_versions: Vec<_> = v_list
        .versions
        .into_iter()
        .filter(|v| match target_version {
            "all" => true,
            "stable" | "lts" | "testing" | "unstable" => {
                v.release_type.to_string().to_lowercase() == target_version
            }
            _ => {
                if target_version.contains('*') {
                    match_version_with_wildcard(&v.version.to_string(), target_version)
                } else {
                    v.version.to_string() == target_version
                }
            }
        })
        .collect();

    // Sort by version descending, then by release_date descending
    filtered_versions.sort_by(|a, b| {
        b.version.cmp(&a.version)
            .then_with(|| b.release_date.cmp(&a.release_date))
    });

    if truncate && !filtered_versions.is_empty() {
        filtered_versions.truncate(1);
    }

    for v in filtered_versions {
        table.add_row(vec![
            repo_name.clone(),
            v.pkgname,
            v.version.to_string(),
            if v.stream.is_empty() {
                "-".to_string()
            } else {
                v.stream
            },
            v.release_date,
            v.release_type.to_string(),
        ]);
    }
}
