use crate::commands::repo::sync;
use crate::models::config::Config;
use crate::models::repository::{Repositories, Repository};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;
use anyhow::{Context, Result};

#[derive(Debug, Serialize, Deserialize)]
struct RepoMetadata {
    /// Name of the repository defined in pi.repo.json
    /// Example: "pi-main"
    name: String,
}

/// Adds a new repository to the pi configuration and performs an initial sync.
/// 
/// Example path: "./my-custom-repo" -> "/home/user/my-custom-repo"
/// Example metadata file: "/home/user/my-custom-repo/pi.repo.json"
pub fn run(config: &Config, path: &str) {
    if let Err(e) = execute_repo_add(config, path) {
        log::error!("failed to add repo: {}", e);
        std::process::exit(1);
    }
}

fn execute_repo_add(config: &Config, path: &str) -> Result<()> {
    let abs_path = fs::canonicalize(path).context("Failed to get absolute path")?;
    let metadata = load_repo_metadata(&abs_path)?;
    
    let mut repo_config = Repositories::load(config).context("Failed to load repositories")?;
    let path_str = abs_path.to_string_lossy().to_string();

    validate_new_repo(&repo_config, &metadata.name, &path_str)?;

    let repo = Repository::new(path_str, metadata.name.clone());
    repo_config.repositories.push(repo);
    repo_config.save(config).context("Failed to save repositories")?;

    log::info!("added repo: {} at {}", metadata.name, abs_path.display());

    // Automatically sync the newly added repository
    sync::run(config, Some(&metadata.name));
    Ok(())
}

/// Loads and parses the pi.repo.json file from the repository path.
fn load_repo_metadata(repo_path: &Path) -> Result<RepoMetadata> {
    let metadata_path = repo_path.join("pi.repo.json");
    if !metadata_path.exists() {
        anyhow::bail!("pi.repo.json missing in {}", repo_path.display());
    }

    let content = fs::read_to_string(&metadata_path)
        .with_context(|| format!("Failed to read {}", metadata_path.display()))?;
    
    serde_json::from_str(&content)
        .with_context(|| format!("Failed to parse {}", metadata_path.display()))
}

/// Checks if the repository already exists by name or path.
fn validate_new_repo(repo_config: &Repositories, name: &str, path: &str) -> Result<()> {
    if repo_config.repositories.iter().any(|r| r.path == path) {
        anyhow::bail!("repository already exists at path: {}", path);
    }

    if repo_config.repositories.iter().any(|r| r.name == name) {
        anyhow::bail!("repository with name '{}' already exists", name);
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;
    use std::fs;

    #[test]
    fn test_execute_repo_add_success() {
        let tmp = tempdir().unwrap();
        let repo_dir = tmp.path().join("my-repo");
        fs::create_dir_all(&repo_dir).unwrap();
        
        let config = Config::new_test(tmp.path().to_path_buf());
        
        let metadata = RepoMetadata { name: "test-repo".to_string() };
        let metadata_content = serde_json::to_string(&metadata).unwrap();
        fs::write(repo_dir.join("pi.repo.json"), &metadata_content).unwrap();

        let result = execute_repo_add(&config, repo_dir.to_str().unwrap());
        assert!(result.is_ok());

        let repo_config = Repositories::load(&config).unwrap();
        assert_eq!(repo_config.repositories.len(), 1);
        assert_eq!(repo_config.repositories[0].name, "test-repo");
    }

    #[test]
    fn test_execute_repo_add_missing_metadata() {
        let tmp = tempdir().unwrap();
        let repo_dir = tmp.path().join("my-repo");
        fs::create_dir_all(&repo_dir).unwrap();
        
        let config = Config::new_test(tmp.path().to_path_buf());

        let result = execute_repo_add(&config, repo_dir.to_str().unwrap());
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("pi.repo.json missing"));
    }

    #[test]
    fn test_execute_repo_add_duplicate() {
        let tmp = tempdir().unwrap();
        let repo_dir = tmp.path().join("my-repo");
        fs::create_dir_all(&repo_dir).unwrap();
        
        let config = Config::new_test(tmp.path().to_path_buf());
        
        let metadata = RepoMetadata { name: "test-repo".to_string() };
        let metadata_content = serde_json::to_string(&metadata).unwrap();
        fs::write(repo_dir.join("pi.repo.json"), &metadata_content).unwrap();

        // First add
        execute_repo_add(&config, repo_dir.to_str().unwrap()).unwrap();

        // Second add (duplicate path)
        let result = execute_repo_add(&config, repo_dir.to_str().unwrap());
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("repository already exists at path"));

        // Duplicate name, different path
        let repo_dir2 = tmp.path().join("my-repo-2");
        fs::create_dir_all(&repo_dir2).unwrap();
        fs::write(repo_dir2.join("pi.repo.json"), &metadata_content).unwrap();
        
        let result = execute_repo_add(&config, repo_dir2.to_str().unwrap());
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("repository with name 'test-repo' already exists"));
    }
}
