use crate::models::config::Config;
use anyhow::Context;
use serde::{Deserialize, Serialize};
use std::fs;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Repository {
    pub path: String,
    pub name: String,
}

impl Repository {
    pub fn new(path: String, name: String) -> Self {
        Self { path, name }
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Repositories {
    pub repositories: Vec<Repository>,
}

impl Repositories {
    pub fn get_all(config: &Config) -> &Self {
        config.state.repositories.get_or_init(|| {
            Self::load(config).unwrap_or_else(|e| {
                log::warn!("failed to load repos: {}", e);
                Self {
                    repositories: Vec::new(),
                }
            })
        })
    }

    pub fn load(config: &Config) -> anyhow::Result<Self> {
        let config_file = config.repositories_file();
        if !config_file.exists() {
            return Ok(Self {
                repositories: Vec::new(),
            });
        }
        let content = fs::read_to_string(&config_file)
            .with_context(|| format!("Failed to read config file: {:?}", config_file))?;
        serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse config file: {:?}", config_file))
    }

    pub fn save(&self, config: &Config) -> anyhow::Result<()> {
        fs::create_dir_all(&config.config_dir).context("Failed to create config directory")?;
        let config_file = config.repositories_file();
        let content = serde_json::to_string_pretty(self).context("Failed to serialize config")?;
        fs::write(&config_file, content)
            .with_context(|| format!("Failed to write config file: {:?}", config_file))
    }
}
