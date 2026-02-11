use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::RepositoryConfig;
use comfy_table::Table;
use std::fs;

pub fn run(config: &Config, name: Option<&str>) {
    let config_file = config.repositories_file();

    if !config_file.exists() {
        println!("No repositories configured.");
        return;
    }

    let content = fs::read_to_string(&config_file).expect("Failed to read config file");
    let repo_config: RepositoryConfig =
        serde_json::from_str(&content).expect("Failed to parse config file");

    let mut table = Table::new();
    table.set_header(vec!["Repo Name", "Type", "Name", "Discover Fn"]);

    for repo in repo_config.repositories {
        if let Some(target_name) = name {
            if repo.name != target_name {
                continue;
            }
        }

        let cache_file = config.package_cache_file(&repo.uuid);
        if cache_file.exists() {
            let content = fs::read_to_string(&cache_file).expect("Failed to read cache file");
            let package_list: PackageList =
                serde_json::from_str(&content).expect("Failed to parse cache file");

            for pkg in package_list.packages {
                table.add_row(vec![
                    repo.name.clone(),
                    "Package".to_string(),
                    pkg.name,
                    pkg.function_name,
                ]);
            }

            for inst in package_list.installers {
                table.add_row(vec![
                    repo.name.clone(),
                    "Installer".to_string(),
                    inst.name,
                    inst.function_name,
                ]);
            }
        }
    }

    println!("{table}");
}

