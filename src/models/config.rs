use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct Config {
    pub cache_dir: PathBuf,
    pub config_dir: PathBuf,
    pub state_dir: PathBuf,
    pub meta_dir: PathBuf,
    pub download_dir: PathBuf,
}

impl Config {
    pub fn new() -> Self {
        let cache_dir = dirs_next::cache_dir()
            .expect("Failed to get cache directory")
            .join("pi");
        let config_dir = dirs_next::config_dir()
            .expect("Failed to get config directory")
            .join("pi");
        let state_dir = dirs_next::data_local_dir()
            .expect("Failed to get local data directory")
            .join("pi");

        let meta_dir = cache_dir.join("meta");
        let download_dir = cache_dir.join("downloads");

        Self {
            cache_dir,
            config_dir,
            state_dir,
            meta_dir,
            download_dir,
        }
    }

    pub fn repositories_file(&self) -> PathBuf {
        self.config_dir.join("repositories.json")
    }

    pub fn package_cache_file(&self, uuid: &str) -> PathBuf {
        self.meta_dir.join(format!("packages-{}.json", uuid))
    }

    pub fn version_cache_file(&self, uuid: &str, safe_name: &str) -> PathBuf {
        self.meta_dir.join(format!("version-{}-{}.json", uuid, safe_name))
    }
}
