use crate::models::config::Config;
use std::fs;

pub fn run(config: &Config) {
    if config.cache_dir.exists() {
        match fs::remove_dir_all(&config.cache_dir) {
            Ok(_) => log::info!("cleaned cache: {}", config.cache_dir.display()),
            Err(e) => log::error!("clean failed: {}", e),
        }
    } else {
        log::debug!("cache missing: {}", config.cache_dir.display());
    }
}
