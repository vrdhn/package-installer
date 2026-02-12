use crate::models::config::Config;
use allocative::Allocative;
use anyhow::Context;
use serde::{Deserialize, Serialize};
use std::fs;
use std::sync::Arc;

#[derive(Debug, Clone, Serialize, Deserialize, Allocative)]
pub enum ManagerCommand {
    Auto,
    Custom(String),
}

impl Default for ManagerCommand {
    fn default() -> Self {
        Self::Auto
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative)]
pub struct VersionEntry {
    pub pkgname: String,
    pub version: String,
    pub release_date: String,
    pub release_type: String,
    pub url: String,
    pub filename: String,
    pub checksum: String,
    pub checksum_url: String,
    pub filemap: std::collections::HashMap<String, String>,
    #[serde(default)]
    pub manager_command: ManagerCommand,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct VersionList {
    pub versions: Vec<VersionEntry>,
}

impl VersionList {
    pub fn get_for_package(
        config: &Config,
        repo: &crate::models::repository::Repository,
        package_name: &str,
        package_entry: Option<&crate::models::package_entry::PackageEntry>,
        manager_entry: Option<(&crate::models::package_entry::ManagerEntry, &str)>,
    ) -> Option<Arc<Self>> {
        let key = format!("{}:{}", repo.uuid, package_name);
        use dashmap::mapref::entry::Entry;

        match config.state.version_lists.entry(key) {
            Entry::Occupied(occupied) => Some(occupied.get().clone()),
            Entry::Vacant(vacant) => {
                // Try to load from disk first
                if let Ok(list) = Self::load(config, &repo.uuid, package_name) {
                    let arc_list = Arc::new(list);
                    return Some(vacant.insert(arc_list).clone());
                }

                // If not on disk, sync
                if let Some(pkg) = package_entry {
                    crate::services::sync::sync_package(config, repo, pkg);
                } else if let Some((mgr, pkg_name)) = manager_entry {
                    crate::services::sync::sync_manager_package(
                        config,
                        repo,
                        mgr,
                        package_name.split(':').next().unwrap_or(""), // manager name
                        pkg_name,
                    );
                }

                // Try to load again after sync
                if let Ok(list) = Self::load(config, &repo.uuid, package_name) {
                    let arc_list = Arc::new(list);
                    return Some(vacant.insert(arc_list).clone());
                }
                None
            }
        }
    }

    pub fn load(config: &Config, repo_uuid: &str, package_name: &str) -> anyhow::Result<Self> {
        let safe_name = package_name.replace('/', "#");
        let cache_file = config.version_cache_file(repo_uuid, &safe_name);
        let content = fs::read_to_string(&cache_file)
            .with_context(|| format!("Failed to read version cache file: {:?}", cache_file))?;
        serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse version cache file: {:?}", cache_file))
    }

    pub fn save(&self, config: &Config, repo_uuid: &str, package_name: &str) -> anyhow::Result<()> {
        fs::create_dir_all(&config.meta_dir).context("Failed to create meta directory")?;
        let safe_name = package_name.replace('/', "#");
        let cache_file = config.version_cache_file(repo_uuid, &safe_name);
        let content =
            serde_json::to_string_pretty(self).context("Failed to serialize version list")?;
        fs::write(&cache_file, content)
            .with_context(|| format!("Failed to write version cache file: {:?}", cache_file))
    }
}
