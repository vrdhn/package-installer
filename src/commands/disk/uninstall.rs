use crate::models::config::Config;
use std::fs;

pub fn run(config: &Config, confirm: bool) {
    if !confirm {
        println!("Please provide the --confirm flag to proceed with uninstallation.");
        println!("This will delete config, state, and cache directories.");
        return;
    }

    let dirs = [
        ("Config", &config.config_dir),
        ("Cache", &config.cache_dir),
        ("State", &config.state_dir),
    ];

    for (name, path) in dirs {
        if path.exists() {
            match fs::remove_dir_all(path) {
                Ok(_) => println!("Successfully removed {} directory: {}", name, path.display()),
                Err(e) => eprintln!("Failed to remove {} directory {}: {}", name, path.display(), e),
            }
        } else {
            println!("{} directory does not exist: {}", name, path.display());
        }
    }

    println!("Uninstallation complete.");
}
