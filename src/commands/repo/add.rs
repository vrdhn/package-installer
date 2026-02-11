use crate::commands::repo::sync;
use crate::models::config::Config;
use crate::models::repository::{Repository, RepositoryConfig};
use std::fs;

pub fn run(config: &Config, path: &str) {
    let abs_path = fs::canonicalize(path).expect("Failed to get absolute path");
    let name = abs_path
        .file_name()
        .expect("Failed to get directory name")
        .to_string_lossy()
        .to_string();
    let abs_path_str = abs_path.to_string_lossy().to_string();

    fs::create_dir_all(&config.config_dir).expect("Failed to create config directory");
    let config_file = config.repositories_file();

    let mut repo_config = if config_file.exists() {
        let content = fs::read_to_string(&config_file).expect("Failed to read config file");
        serde_json::from_str(&content).expect("Failed to parse config file")
    } else {
        RepositoryConfig {
            repositories: Vec::new(),
        }
    };

    if repo_config.repositories.iter().any(|r| r.path == abs_path_str) {
        println!("Repository already exists: {}", abs_path_str);
        return;
    }

    let repo = Repository::new(abs_path_str, name.clone());
    repo_config.repositories.push(repo);

    let content = serde_json::to_string_pretty(&repo_config).expect("Failed to serialize config");
    fs::write(&config_file, content).expect("Failed to write config file");

    println!("Added repository: {} at {}", name, abs_path.display());

    // Automatically call sync
    sync::run(config, Some(&name));
}
