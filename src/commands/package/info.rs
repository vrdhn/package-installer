use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::models::version_entry::VersionList;
use comfy_table::{Table, Attribute, Cell, Color, presets::NOTHING};

pub fn run(config: &Config, selector_str: &str) {
    let selector = match PackageSelector::parse(selector_str) {
        Some(s) => s,
        None => {
            println!("Invalid package selector: {}", selector_str);
            return;
        }
    };

    let repo_config = Repositories::get_all(config);
    let mut found = false;

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
            for pkg in &pkg_list.packages {
                if !selector.package.is_empty() && selector.package != "*" {
                    if pkg.name != selector.package {
                        continue;
                    }
                }

                if let Some(v_list) = VersionList::get_for_package(config, repo, &pkg.name, Some(pkg), None) {
                    if print_package_info(repo.name.clone(), &pkg.name, (*v_list).clone(), target_version) {
                        found = true;
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
                    if print_package_info(repo.name.clone(), &full_name, (*v_list).clone(), target_version) {
                        found = true;
                    }
                }
            }
        }
    }

    if !found {
        println!("No packages found matching: {}", selector_str);
    }
}

fn print_package_info(repo_name: String, _pkg_name: &str, v_list: VersionList, target_version: &str) -> bool {
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

    if filtered_versions.is_empty() {
        return false;
    }

    // Sort by release_date descending
    filtered_versions.sort_by(|a, b| b.release_date.cmp(&a.release_date));

    for v in filtered_versions {
        let mut table = Table::new();
        table.load_preset(NOTHING);
        table.set_header(vec![
            Cell::new("Field").add_attribute(Attribute::Bold),
            Cell::new("Value").add_attribute(Attribute::Bold),
        ]);

        table.add_row(vec!["Repository", &repo_name]);
        table.add_row(vec!["Package", &v.pkgname]);
        table.add_row(vec!["Version", &v.version]);
        table.add_row(vec!["Release Date", &v.release_date]);
        table.add_row(vec!["Release Type", &v.release_type]);
        table.add_row(vec!["Filename", &v.filename]);
        table.add_row(vec!["Checksum", &v.checksum]);
        
        let url_cell = Cell::new(&v.url).fg(Color::Cyan);
        table.add_row(vec![Cell::new("URL"), url_cell]);

        if !v.checksum_url.is_empty() {
            table.add_row(vec!["Checksum URL", &v.checksum_url]);
        }

        if !v.filemap.is_empty() {
            let mut filemap_str = String::new();
            for (src, dest) in &v.filemap {
                if !filemap_str.is_empty() {
                    filemap_str.push('\n');
                }
                filemap_str.push_str(&format!("{} -> {}", src, dest));
            }
            table.add_row(vec!["Filemap", &filemap_str]);
        }

        if !v.env.is_empty() {
            let mut env_str = String::new();
            let mut sorted_keys: Vec<_> = v.env.keys().collect();
            sorted_keys.sort();
            for k in sorted_keys {
                if !env_str.is_empty() {
                    env_str.push('\n');
                }
                env_str.push_str(&format!("{}={}", k, v.env.get(k).unwrap()));
            }
            table.add_row(vec!["Environment", &env_str]);
        }

        match v.manager_command {
            crate::models::version_entry::ManagerCommand::Custom(ref cmd) => {
                table.add_row(vec!["Install Command", cmd]);
            }
            crate::models::version_entry::ManagerCommand::Auto => {
                table.add_row(vec!["Install Command", "Auto"]);
            }
        }

        println!("{table}
");
    }

    true
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
