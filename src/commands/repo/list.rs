use crate::models::repository::RepositoryConfig;
use crate::models::package_entry::PackageList;
use comfy_table::Table;
use std::fs;

pub fn run(name: Option<&str>) {
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
    table.set_header(vec!["Repo Name", "Regexp", "Discover Fn"]);

    for repo in config.repositories {
        if let Some(target_name) = name {
            if repo.name != target_name {
                continue;
            }
        }

        let cache_file = cache_dir.join(format!("packages-{}.json", repo.uuid));
        if cache_file.exists() {
            let content = fs::read_to_string(&cache_file).expect("Failed to read cache file");
            let package_list: PackageList = serde_json::from_str(&content).expect("Failed to parse cache file");

            for pkg in package_list.packages {
                table.add_row(vec![
                    repo.name.clone(),
                    pkg.name,
                    pkg.function_name,
                ]);
            }
        }
    }

    println!("{table}");
}
