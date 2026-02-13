use crate::models::config::Config;
use std::fs;

pub fn run(config: &Config, confirm: bool) {
    if !confirm {
        log::warn!("needs --confirm to delete config, state, and cache");
        return;
    }

    let dirs = [
        ("config", &config.config_dir),
        ("cache", &config.cache_dir),
        ("state", &config.state_dir),
    ];

    for (name, path) in dirs {
        if path.exists() {
            match fs::remove_dir_all(path) {
                Ok(_) => log::info!("removed {}: {}", name, path.display()),
                Err(e) => log::error!("failed to remove {} {}: {}", name, path.display(), e),
            }
        } else {
            log::debug!("{} missing: {}", name, path.display());
        }
    }

    log::info!("uninstall complete");
}
