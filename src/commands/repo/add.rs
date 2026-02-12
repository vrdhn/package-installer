use crate::commands::repo::sync;
use crate::models::config::Config;
use crate::models::repository::{Repository, Repositories};
use std::fs;

pub fn run(config: &Config, path: &str) {
    let abs_path = fs::canonicalize(path).expect("Failed to get absolute path");
    let name = abs_path
        .file_name()
        .expect("Failed to get directory name")
        .to_string_lossy()
        .to_string();
    let abs_path_str = abs_path.to_string_lossy().to_string();

    let mut repo_config = Repositories::load(config).expect("Failed to load repositories");

    if repo_config.repositories.iter().any(|r| r.path == abs_path_str) {
        println!("Repository already exists: {}", abs_path_str);
        return;
    }

    let repo = Repository::new(abs_path_str, name.clone());
    repo_config.repositories.push(repo);

    repo_config.save(config).expect("Failed to save repositories");

    println!("Added repository: {} at {}", name, abs_path.display());

    // Automatically call sync
    sync::run(config, Some(&name));
}
