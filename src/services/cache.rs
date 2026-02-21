use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::time::{Duration, SystemTime};

pub struct Cache {
    dir: PathBuf,
    ttl: Duration,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct StepResult {
    pub step_hash: String,
    pub timestamp: String,
    pub output_path: Option<PathBuf>,
    pub status: String,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct PackageBuildCache {
    pub versions: HashMap<String, Vec<StepResult>>,
}

pub struct BuildCache {
    cache_dir: PathBuf,
}

impl BuildCache {
    pub fn new(cache_dir: PathBuf) -> Self {
        let dir = cache_dir.join("builds");
        if !dir.exists() {
            let _ = fs::create_dir_all(&dir);
        }
        Self { cache_dir: dir }
    }

    fn get_file_path(&self, pkgname: &str) -> PathBuf {
        let safe_name = pkgname.replace(['/', '\\', ' ', ':'], "_");
        self.cache_dir.join(format!("{}.json", safe_name))
    }

    pub fn load(&self, pkgname: &str) -> PackageBuildCache {
        let path = self.get_file_path(pkgname);
        if let Ok(content) = fs::read_to_string(path) {
            if let Ok(cache) = serde_json::from_str(&content) {
                return cache;
            }
        }
        PackageBuildCache::default()
    }

    pub fn save(&self, pkgname: &str, cache: &PackageBuildCache) -> Result<()> {
        let path = self.get_file_path(pkgname);
        let content = serde_json::to_string_pretty(cache)?;
        fs::write(path, content)?;
        Ok(())
    }

    pub fn get_step_result(&self, pkgname: &str, version: &str, step_index: usize, step_hash: &str) -> Option<StepResult> {
        let cache = self.load(pkgname);
        if let Some(steps) = cache.versions.get(version) {
            if let Some(result) = steps.get(step_index) {
                if result.step_hash == step_hash && result.status == "Success" {
                    return Some(result.clone());
                }
            }
        }
        None
    }

    pub fn update_step_result(&self, pkgname: &str, version: &str, step_index: usize, result: StepResult) -> Result<()> {
        let mut cache = self.load(pkgname);
        let steps = cache.versions.entry(version.to_string()).or_default();
        
        if step_index < steps.len() {
            steps[step_index] = result;
        } else if step_index == steps.len() {
            steps.push(result);
        } else {
            // Fill gaps if needed, though pipeline is usually sequential
            while steps.len() < step_index {
                steps.push(StepResult {
                    step_hash: "unknown".to_string(),
                    timestamp: "".to_string(),
                    output_path: None,
                    status: "Skipped".to_string(),
                });
            }
            steps.push(result);
        }

        self.save(pkgname, &cache)
    }
}

impl Cache {
    pub fn new(dir: PathBuf, ttl: Duration) -> Self {
        Self { dir, ttl }
    }

    pub fn get_path(&self, url: &str) -> PathBuf {
        let sanitized = url
            .replace("://", "_")
            .replace("/", "_")
            .replace(":", "_")
            .replace("?", "_")
            .replace("&", "_")
            .replace("=", "_");
        self.dir.join(sanitized)
    }

    pub fn read(&self, url: &str) -> Result<Option<String>> {
        let path = self.get_path(url);
        if !path.exists() {
            return Ok(None);
        }

        let metadata = fs::metadata(&path)?;
        let modified = metadata.modified()?;
        if SystemTime::now().duration_since(modified)? > self.ttl {
            return Ok(None);
        }

        let content = fs::read_to_string(path)?;
        Ok(Some(content))
    }

    pub fn write(&self, url: &str, content: &str) -> Result<()> {
        if !self.dir.exists() {
            fs::create_dir_all(&self.dir)?;
        }
        let path = self.get_path(url);
        fs::write(path, content)?;
        Ok(())
    }
}
