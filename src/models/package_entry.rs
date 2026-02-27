use crate::models::config::Config;
use allocative::Allocative;
use anyhow::Context;
use serde::{Deserialize, Serialize};
use std::fs;
use std::sync::Arc;
use std::collections::HashMap;

#[derive(Debug, Clone, Allocative, Serialize, Deserialize)]
pub struct RegistryEntry {
    pub name: String,
    pub function_name: String,
    pub filename: String,
}

// Aliases for compatibility
pub type PackageEntry = RegistryEntry;
pub type ManagerEntry = RegistryEntry;

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct PackageList {
    pub packages: HashMap<String, RegistryEntry>,
    pub managers: HashMap<String, RegistryEntry>,
}

impl PackageList {
    pub fn get_for_repo(config: &Config, repo: &crate::models::repository::Repository, force: bool) -> Option<Arc<Self>> {
        use dashmap::mapref::entry::Entry;

        // Check cache first using DashMap for thread-safe concurrent access.
        if !config.force && !force {
            if let Entry::Occupied(occupied) = config.state.package_lists.entry(repo.name.clone()) {
                return Some(occupied.get().clone());
            }

            // Try to load from disk if not forcing
            if let Ok(list) = Self::load(config, &repo.name) {
                let arc_list = Arc::new(list);
                return Some(config.state.package_lists.entry(repo.name.clone()).or_insert(arc_list).clone());
            }
        }

        // If force is true, or if not found on disk, sync
        log::info!("[{}] {}syncing", repo.name, if config.force || force { "force " } else { "" });
        if let Err(e) = crate::services::sync::sync_repo(config, repo) {
            log::error!("[{}] sync failed: {}", repo.name, e);
            return None;
        }

        // Try to load again after sync
        if let Ok(list) = Self::load(config, &repo.name) {
            let arc_list = Arc::new(list);
            return Some(config.state.package_lists.entry(repo.name.clone()).or_insert(arc_list).clone());
        }
        None
    }

    pub fn load(config: &Config, repo_name: &str) -> anyhow::Result<Self> {
        let cache_file = config.package_cache_file(repo_name);
        let content = fs::read_to_string(&cache_file)
            .with_context(|| format!("Failed to read package cache file: {:?}", cache_file))?;
        serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse package cache file: {:?}", cache_file))
    }

    pub fn save(&self, config: &Config, repo_name: &str) -> anyhow::Result<()> {
        fs::create_dir_all(&config.cache_meta_dir).context("Failed to create meta directory")?;
        let cache_file = config.package_cache_file(repo_name);
        let content =
            serde_json::to_string_pretty(self).context("Failed to serialize package list")?;
        fs::write(&cache_file, content)
            .with_context(|| format!("Failed to write package cache file: {:?}", cache_file))
    }
}
