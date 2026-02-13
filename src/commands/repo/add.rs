use crate::commands::repo::sync;
use crate::models::config::Config;
use crate::models::repository::{Repositories, Repository};
use serde::{Deserialize, Serialize};
use std::fs;

#[derive(Debug, Serialize, Deserialize)]
struct RepoMetadata {
    name: String,
}

pub fn run(config: &Config, path: &str) {
    let abs_path = fs::canonicalize(path).expect("Failed to get absolute path");
    let pi_repo_path = abs_path.join("pi.repo.json");

    if !pi_repo_path.exists() {
        log::error!("pi.repo.json missing in {}", abs_path.display());
        std::process::exit(1);
    }

    let metadata_content =
        fs::read_to_string(&pi_repo_path).expect("Failed to read pi.repo.json");
    let metadata: RepoMetadata =
        serde_json::from_str(&metadata_content).expect("Failed to parse pi.repo.json");

    let name = metadata.name;
    let abs_path_str = abs_path.to_string_lossy().to_string();

    let mut repo_config = Repositories::load(config).expect("Failed to load repositories");

    if repo_config
        .repositories
        .iter()
        .any(|r| r.path == abs_path_str)
    {
        log::warn!("repo exists at {}", abs_path_str);
        return;
    }

    if repo_config.repositories.iter().any(|r| r.name == name) {
        log::error!("repo {} already exists", name);
        std::process::exit(1);
    }

    let repo = Repository::new(abs_path_str, name.clone());
    repo_config.repositories.push(repo);

    repo_config
        .save(config)
        .expect("Failed to save repositories");

    log::info!("added repo: {} at {}", name, abs_path.display());

    // Automatically call sync
    sync::run(config, Some(&name));
}
