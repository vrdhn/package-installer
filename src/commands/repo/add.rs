use crate::commands::repo::sync;
use crate::models::repository::{Repository, RepositoryConfig};
use std::fs;

pub fn run(path: &str) {
    let abs_path = fs::canonicalize(path).expect("Failed to get absolute path");
    let name = abs_path
        .file_name()
        .expect("Failed to get directory name")
        .to_string_lossy()
        .to_string();
    let abs_path_str = abs_path.to_string_lossy().to_string();

    let config_dir = dirs_next::config_dir()
        .expect("Failed to get config directory")
        .join("pi");
    fs::create_dir_all(&config_dir).expect("Failed to create config directory");
    let config_file = config_dir.join("repositories.json");

    let mut config = if config_file.exists() {
        let content = fs::read_to_string(&config_file).expect("Failed to read config file");
        serde_json::from_str(&content).expect("Failed to parse config file")
    } else {
        RepositoryConfig {
            repositories: Vec::new(),
        }
    };

    if config.repositories.iter().any(|r| r.path == abs_path_str) {
        println!("Repository already exists: {}", abs_path_str);
        return;
    }

    let repo = Repository::new(abs_path_str, name.clone());
    config.repositories.push(repo);

    let content = serde_json::to_string_pretty(&config).expect("Failed to serialize config");
    fs::write(&config_file, content).expect("Failed to write config file");

    println!("Added repository: {} at {}", name, abs_path.display());

    // Automatically call sync
    sync::run(Some(&name));
}
