use anyhow::Result;
use std::path::PathBuf;
use std::fs;
use std::time::{SystemTime, Duration};
use sha2::{Sha256, Digest};
use hex;

pub struct Cache {
    dir: PathBuf,
    ttl: Duration,
}

impl Cache {
    pub fn new(dir: PathBuf, ttl: Duration) -> Self {
        Self { dir, ttl }
    }

    pub fn get_path(&self, url: &str) -> PathBuf {
        let mut hasher = Sha256::new();
        hasher.update(url.as_bytes());
        let hash = hex::encode(hasher.finalize());
        self.dir.join(hash)
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
