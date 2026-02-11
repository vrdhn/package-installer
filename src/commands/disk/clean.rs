use crate::models::config::Config;
use std::fs;

pub fn run(config: &Config) {
    if config.cache_dir.exists() {
        match fs::remove_dir_all(&config.cache_dir) {
            Ok(_) => println!("Successfully cleaned cache directory: {}", config.cache_dir.display()),
            Err(e) => eprintln!("Failed to clean cache directory: {}", e),
        }
    } else {
        println!("Cache directory does not exist: {}", config.cache_dir.display());
    }
}
