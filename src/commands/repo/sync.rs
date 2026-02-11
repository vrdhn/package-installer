use crate::commands::repo::list;
use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::{Repository, Repositories};
use crate::starlark::runtime::evaluate_file;
use rayon::prelude::*;
use std::fs;
use std::path::Path;
use walkdir::WalkDir;

pub fn run(config: &Config, name: Option<&str>) {
    let config_file = config.repositories_file();

    if !config_file.exists() {
        println!("No repositories configured.");
        return;
    }

    let content = fs::read_to_string(&config_file).expect("Failed to read config file");
    let repo_config: Repositories =
        serde_json::from_str(&content).expect("Failed to parse config file");

    fs::create_dir_all(&config.meta_dir).expect("Failed to create cache directory");

    repo_config.repositories.par_iter().for_each(|repo| {
        if let Some(target_name) = name {
            if repo.name != target_name {
                return;
            }
        }

        sync_repo(config, repo);
    });

    list::run(config, name);
}

fn sync_repo(config: &Config, repo: &Repository) {
    println!("Syncing repository: {}...", repo.name);
    let mut all_packages = Vec::new();
    let mut all_installers = Vec::new();
    let repo_path = Path::new(&repo.path);

    for entry in WalkDir::new(repo_path)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| e.path().extension().map_or(false, |ext| ext == "star"))
    {
        let star_file_path = entry.path();
        match evaluate_file(star_file_path, config.download_dir.clone()) {
            Ok((packages, installers)) => {
                let rel_path = star_file_path
                    .strip_prefix(repo_path)
                    .unwrap_or(star_file_path)
                    .to_string_lossy()
                    .to_string();

                for mut pkg in packages {
                    pkg.filename = rel_path.clone();
                    all_packages.push(pkg);
                }
                for mut inst in installers {
                    inst.filename = rel_path.clone();
                    all_installers.push(inst);
                }
            }
            Err(e) => {
                eprintln!("Error evaluating {}: {}", star_file_path.display(), e);
            }
        }
    }

    let package_list = PackageList {
        packages: all_packages,
        installers: all_installers,
    };
    let cache_file = config.package_cache_file(&repo.uuid);
    let content =
        serde_json::to_string_pretty(&package_list).expect("Failed to serialize package list");
    fs::write(&cache_file, content).expect("Failed to write cache file");
    println!(
        "Synced {} packages and {} installers for {}",
        package_list.packages.len(),
        package_list.installers.len(),
        repo.name
    );
}
