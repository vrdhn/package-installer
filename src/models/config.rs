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
    pub cache_meta_dir: PathBuf,
    pub cache_download_dir: PathBuf,
    pub cache_packages_dir: PathBuf,
    pub cache_pilocals_dir: PathBuf,
    pub force: bool,
    pub state: Arc<State>,
}

#[derive(Debug, Default)]
pub struct State {
    pub repositories: OnceLock<Repositories>,
    /// Thread-safe cache of package lists for each repository.
    /// Uses DashMap to allow concurrent read/write access across Starlark evaluations.
    /// Keyed by repository name.
    pub package_lists: DashMap<String, Arc<PackageList>>,
    /// Thread-safe cache of version lists for each package.
    /// Keyed by "repo_name:package_name".
    pub version_lists: DashMap<String, Arc<VersionList>>,
    /// Per-URL download locks to prevent redundant concurrent downloads of the same resource.
    /// The Mutex is only held during the actual network transfer.
    /// Keyed by resource URL.
    pub download_locks: DashMap<String, Arc<parking_lot::Mutex<()>>>,
}

impl Config {
    pub fn new(force: bool) -> Self {
        let xdg = xdg::BaseDirectories::with_prefix("pi");

        let cache_dir = xdg.get_cache_home().expect("Failed to get cache home");
        let config_dir = xdg.get_config_home().expect("Failed to get config home");
        let state_dir = xdg.get_state_home().expect("Failed to get state home");

        let meta_dir = xdg.create_cache_directory("meta")
	    .expect("Failed to create meta directory");
        let download_dir = xdg.create_cache_directory("downloads")
	    .expect("Failed to create downloads directory");
        let packages_dir = xdg.create_cache_directory("packages")
	    .expect("Failed to create packages directory");
        let pilocals_dir = xdg.create_cache_directory("pilocals")
	    .expect("Failed to create pilocals directory");

        Self {
            cache_dir,
            config_dir,
            state_dir,
            cache_meta_dir: meta_dir,
            cache_download_dir: download_dir,
            cache_packages_dir: packages_dir,
            cache_pilocals_dir: pilocals_dir,
            force,
            state: Arc::new(State::default()),
        }
    }

    pub fn new_test(base_dir: PathBuf) -> Self {
        let cache_dir = base_dir.join("cache");
        let config_dir = base_dir.join("config");
        let state_dir = base_dir.join("state");
        let meta_dir = cache_dir.join("meta");
        let download_dir = cache_dir.join("downloads");
        let packages_dir = cache_dir.join("packages");
        let pilocals_dir = cache_dir.join("pilocals");

        std::fs::create_dir_all(&cache_dir).unwrap();
        std::fs::create_dir_all(&config_dir).unwrap();
        std::fs::create_dir_all(&state_dir).unwrap();
        std::fs::create_dir_all(&meta_dir).unwrap();
        std::fs::create_dir_all(&download_dir).unwrap();
        std::fs::create_dir_all(&packages_dir).unwrap();
        std::fs::create_dir_all(&pilocals_dir).unwrap();

        Self {
            cache_dir,
            config_dir,
            state_dir,
            cache_meta_dir: meta_dir,
            cache_download_dir: download_dir,
            cache_packages_dir: packages_dir,
            cache_pilocals_dir: pilocals_dir,
            force: false,
            state: Arc::new(State::default()),
        }
    }

    pub fn repositories_file(&self) -> PathBuf {
        self.config_dir.join("repositories.json")
    }

    pub fn package_cache_file(&self, repo_name: &str) -> PathBuf {
        self.cache_meta_dir.join(format!("packages-{}.json", repo_name))
    }

    pub fn version_cache_file(&self, repo_name: &str, safe_name: &str) -> PathBuf {
        self.cache_meta_dir.join(format!("version-{}-{}.json", repo_name, safe_name))
    }

    pub fn get_user(&self) -> String {
        whoami::username()
    }

    pub fn get_hostname(&self) -> String {
        whoami::fallible::hostname().unwrap_or_else(|_| "pi-cave".to_string())
    }

    pub fn get_host_home(&self) -> PathBuf {
        dirs_next::home_dir().expect("Failed to get home directory")
    }

    pub fn is_inside_cave(&self) -> bool {
        std::env::var("PI_CAVE").is_ok()
    }

    pub fn pilocal_path(&self, cave_name: &str, _variant: Option<&str>) -> PathBuf {
        self.cache_pilocals_dir.join(cave_name)
    }
}
