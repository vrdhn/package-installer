use crate::models::repository::{Repository, RepositoryConfig};
use crate::models::package_entry::PackageList;
use crate::starlark::runtime::evaluate_file;
use crate::commands::repo::list;
use std::fs;
use std::path::Path;
use walkdir::WalkDir;

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
    fs::create_dir_all(&cache_dir).expect("Failed to create cache directory");

    let download_dir = cache_dir.join("downloads");

    for repo in config.repositories {
        if let Some(target_name) = name {
            if repo.name != target_name {
                continue;
            }
        }

        sync_repo(&repo, &cache_dir, &download_dir);
    }

    list::run(name);
}

fn sync_repo(repo: &Repository, cache_dir: &Path, download_dir: &Path) {
    println!("Syncing repository: {}...", repo.name);
    let mut all_packages = Vec::new();
    let repo_path = Path::new(&repo.path);

    for entry in WalkDir::new(repo_path)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| e.path().extension().map_or(false, |ext| ext == "star"))
    {
        let star_file_path = entry.path();
        match evaluate_file(star_file_path, download_dir.to_path_buf()) {
            Ok(packages) => {
                for mut pkg in packages {
                    // Make filename relative to repo path
                    if let Ok(rel_path) = star_file_path.strip_prefix(repo_path) {
                        pkg.filename = rel_path.to_string_lossy().to_string();
                    }
                    all_packages.push(pkg);
                }
            }
            Err(e) => {
                eprintln!("Error evaluating {}: {}", star_file_path.display(), e);
            }
        }
    }

    let package_list = PackageList { packages: all_packages };
    let cache_file = cache_dir.join(format!("packages-{}.json", repo.uuid));
    let content = serde_json::to_string_pretty(&package_list).expect("Failed to serialize package list");
    fs::write(&cache_file, content).expect("Failed to write cache file");
    println!("Synced {} packages for {}", package_list.packages.len(), repo.name);
}
