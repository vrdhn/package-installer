use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use crate::models::version_entry::VersionList;
use dashmap::DashMap;
use std::path::PathBuf;
use std::sync::{Arc, OnceLock};

#[derive(Debug, Clone)]
pub struct Config {
    pub cache_dir: PathBuf,
    pub config_dir: PathBuf,
    pub state_dir: PathBuf,
    pub meta_dir: PathBuf,
    pub download_dir: PathBuf,
    pub packages_dir: PathBuf,
    pub pilocals_dir: PathBuf,
    pub state: Arc<State>,
}

#[derive(Debug, Default)]
pub struct State {
    pub repositories: OnceLock<Repositories>,
    pub package_lists: DashMap<String, Arc<PackageList>>,
    pub version_lists: DashMap<String, Arc<VersionList>>,
    pub download_locks: DashMap<String, Arc<parking_lot::Mutex<()>>>,
    pub meta_dir: PathBuf,
    pub download_dir: PathBuf,
    pub packages_dir: PathBuf,
    pub pilocals_dir: PathBuf,
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
        let packages_dir = cache_dir.join("packages");
        let pilocals_dir = cache_dir.join("pilocals");

        Self {
            cache_dir,
            config_dir,
            state_dir,
            meta_dir: meta_dir.clone(),
            download_dir: download_dir.clone(),
            packages_dir: packages_dir.clone(),
            pilocals_dir: pilocals_dir.clone(),
            state: Arc::new(State {
                meta_dir,
                download_dir,
                packages_dir,
                pilocals_dir,
                ..Default::default()
            }),
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

    pub fn get_user(&self) -> String {
        whoami::username()
    }

    pub fn get_host_home(&self) -> PathBuf {
        dirs_next::home_dir().expect("Failed to get home directory")
    }

    pub fn is_inside_cave(&self) -> bool {
        std::env::var("PI_CAVE").is_ok()
    }

    pub fn pilocal_path(&self, cave_name: &str, variant: Option<&str>) -> PathBuf {
        let name = if let Some(v) = variant {
            let v = v.strip_prefix(':').unwrap_or(v);
            format!("{}-{}", cave_name, v)
        } else {
            cave_name.to_string()
        };
        self.pilocals_dir.join(name)
    }
}
