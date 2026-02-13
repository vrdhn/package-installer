use anyhow::Result;
use std::fs;
use std::path::PathBuf;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use crate::services::db::Db;
use std::sync::Arc;

pub struct Cache {
    dir: PathBuf,
    ttl: Duration,
    db: Arc<Db>,
}

impl Cache {
    pub fn new(dir: PathBuf, ttl: Duration, db: Arc<Db>) -> Self {
        Self { dir, ttl, db }
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
        
        // Check DB first
        if let Some((cached_path, expires)) = self.db.get_cache_metadata(url)? {
            let now = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
            if now < expires && PathBuf::from(cached_path).exists() {
                let content = fs::read_to_string(path)?;
                return Ok(Some(content));
            }
        }

        // Fallback to file-based check for legacy or missing DB entries
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
        fs::write(&path, content)?;

        let expires = SystemTime::now()
            .duration_since(UNIX_EPOCH)?
            .as_secs() + self.ttl.as_secs();
        
        self.db.set_cache_metadata(url, &path.to_string_lossy(), expires)?;
        
        Ok(())
    }
}
