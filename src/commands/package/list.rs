use crate::models::repository::RepositoryConfig;
use crate::models::package_entry::PackageList;
use crate::models::version_entry::VersionList;
use crate::models::selector::PackageSelector;
use comfy_table::Table;
use std::fs;
use glob::Pattern;

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

    let mut table = Table::new();
    table.set_header(vec!["Repo", "Package", "Version", "Date", "Type"]);

    let target_version = selector.as_ref().and_then(|s| s.version.clone()).unwrap_or_else(|| "stable".to_string());

    for repo in &config.repositories {
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
            if let Some(ref s) = selector {
                if !s.package.is_empty() && s.package != "*" {
                    if !pkg.name.contains(&s.package) {
                        continue;
                    }
                }
            }

            let version_cache_file = cache_dir.join(format!("version-{}-{}.json", repo.uuid, pkg.name));
            if !version_cache_file.exists() {
                continue;
            }

            let v_content = fs::read_to_string(&version_cache_file).expect("Failed to read version cache file");
            let v_list: VersionList = serde_json::from_str(&v_content).expect("Failed to parse version cache file");

            let mut filtered_versions: Vec<_> = v_list.versions.into_iter().filter(|v| {
                match target_version.as_str() {
                    "latest" => true, // Take all, will sort later
                    "stable" | "lts" | "testing" | "unstable" => v.release_type.to_lowercase() == target_version,
                    _ => {
                        if target_version.contains('*') || target_version.contains('?') {
                            if let Ok(pattern) = Pattern::new(&target_version) {
                                pattern.matches(&v.version)
                            } else {
                                v.version == target_version
                            }
                        } else {
                            v.version == target_version
                        }
                    }
                }
            }).collect();

            // Sort by release_date descending
            filtered_versions.sort_by(|a, b| b.release_date.cmp(&a.release_date));

            for v in filtered_versions.into_iter().take(5) {
                table.add_row(vec![
                    repo.name.clone(),
                    v.pkgname,
                    v.version,
                    v.release_date,
                    v.release_type,
                ]);
            }
        }
    }

    println!("{table}");
}
