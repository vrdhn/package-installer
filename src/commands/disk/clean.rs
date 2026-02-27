use crate::models::config::Config;
use std::fs;

pub fn run(config: &Config, meta: bool, pilocals: bool, packages: bool, downloads: bool, config_flag: bool, state: bool, confirm: bool) {
    if !meta && !pilocals && !packages && !downloads && !config_flag && !state {
        println!("No cleaning flags provided. Specify what to clean:");
        println!("  --meta      Delete package list cache");
        println!("  --pilocals  Delete pilocal cave environments");
        println!("  --packages  Delete downloaded packages");
        println!("  --downloads Delete original downloads");
        println!("  --config    Delete config directory (requires --confirm)");
        println!("  --state     Delete state directory (requires --confirm)");
        return;
    }

    if (config_flag || state) && !confirm {
        log::error!("--config and --state require the --confirm flag to proceed");
        return;
    }

    if meta {
        clean_dir("meta", &config.cache_meta_dir);
    }
    if pilocals {
        clean_dir("pilocals", &config.cache_pilocals_dir);
    }
    if packages {
        clean_dir("packages", &config.cache_packages_dir);
    }
    if downloads {
        clean_dir("downloads", &config.cache_download_dir);
    }
    if config_flag {
        clean_dir("config", &config.config_dir);
    }
    if state {
        clean_dir("state", &config.state_dir);
    }
}

fn clean_dir(name: &str, path: &std::path::Path) {
    if path.exists() {
        match fs::remove_dir_all(path) {
            Ok(_) => log::info!("cleaned {}: {}", name, path.display()),
            Err(e) => log::error!("failed to clean {} {}: {}", name, path.display(), e),
        }
    }
}
