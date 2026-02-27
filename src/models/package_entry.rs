use crate::models::config::Config;
use allocative::Allocative;
use anyhow::Context;
use serde::{Deserialize, Serialize};
use std::fs;
use std::sync::Arc;

#[derive(Debug, Clone, Allocative, Serialize, Deserialize)]
pub struct PackageEntry {
    pub name: String,
    pub function_name: String,
    pub filename: String,
}

#[derive(Debug, Clone, Allocative, Serialize, Deserialize)]
pub struct ManagerEntry {
    pub name: String,
    pub function_name: String,
    pub filename: String,
}

use std::collections::HashMap;

#[derive(Debug, Serialize, Deserialize)]
pub struct PackageList {
    pub packages: Vec<PackageEntry>,
    pub managers: Vec<ManagerEntry>,
    #[serde(skip)]
    pub package_map: HashMap<String, PackageEntry>,
    #[serde(skip)]
    pub manager_map: HashMap<String, ManagerEntry>,
}

impl PackageList {
    pub fn get_for_repo(config: &Config, repo: &crate::models::repository::Repository, force: bool) -> Option<Arc<Self>> {
        use dashmap::mapref::entry::Entry;

        if !config.force && !force {
            if let Entry::Occupied(occupied) = config.state.package_lists.entry(repo.name.clone()) {
                return Some(occupied.get().clone());
            }

            // Try to load from disk if not forcing
            if let Ok(mut list) = Self::load(config, &repo.name) {
                list.initialize_maps();
                let arc_list = Arc::new(list);
                return Some(config.state.package_lists.entry(repo.name.clone()).or_insert(arc_list).clone());
            }
        }

        // If force is true, or if not found on disk, sync
        log::info!("[{}] {}syncing", repo.name, if config.force || force { "force " } else { "" });
        crate::services::sync::sync_repo(config, repo);

        // Try to load again after sync
        if let Ok(mut list) = Self::load(config, &repo.name) {
            list.initialize_maps();
            let arc_list = Arc::new(list);
            return Some(config.state.package_lists.entry(repo.name.clone()).or_insert(arc_list).clone());
        }
        None
    }

    pub fn initialize_maps(&mut self) {
        for pkg in &self.packages {
            self.package_map.insert(pkg.name.clone(), pkg.clone());
        }
        for mgr in &self.managers {
            self.manager_map.insert(mgr.name.clone(), mgr.clone());
        }
    }

    pub fn load(config: &Config, repo_name: &str) -> anyhow::Result<Self> {
        let cache_file = config.package_cache_file(repo_name);
        let content = fs::read_to_string(&cache_file)
            .with_context(|| format!("Failed to read package cache file: {:?}", cache_file))?;
        let mut list: Self = serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse package cache file: {:?}", cache_file))?;
        list.initialize_maps();
        Ok(list)
    }

    pub fn save(&self, config: &Config, repo_name: &str) -> anyhow::Result<()> {
        fs::create_dir_all(&config.meta_dir).context("Failed to create meta directory")?;
        let cache_file = config.package_cache_file(repo_name);
        let content =
            serde_json::to_string_pretty(self).context("Failed to serialize package list")?;
        fs::write(&cache_file, content)
            .with_context(|| format!("Failed to write package cache file: {:?}", cache_file))
    }
}
