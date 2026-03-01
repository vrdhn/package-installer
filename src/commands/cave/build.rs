use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;
use std::collections::HashMap;
use anyhow::{Context, Result};

pub fn run(config: &Config, variant: Option<String>) {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (path, cave) = match Cave::find_in_ancestry(&current_dir) {
        Some(res) => res,
        None => {
            log::error!("no cave found");
            return;
        }
    };

    let variant_str = variant.as_deref().and_then(|v| if v.starts_with(':') { Some(v) } else { None });

    if let Err(e) = execute_build(config, &cave, variant_str) {
        log::error!("build failed: {}", e);
        std::process::exit(1);
    }
}

pub fn execute_build(config: &Config, cave: &Cave, variant: Option<&str>) -> Result<HashMap<String, String>> {
    let settings = cave.get_effective_settings(variant).context("Failed to get effective cave settings")?;
    
    let pilocal_dir = config.pilocal_path(&cave.name, variant);
    let env_cache_file = pilocal_dir.join("env.json");

    if !config.force && !config.rebuild && env_cache_file.exists() {
        let mut cache_valid = true;
        
        // Invalidate if cave configuration changed
        if let Ok(cave_meta) = std::fs::metadata(cave.workspace.join(Cave::FILENAME)) {
            if let Ok(cache_meta) = std::fs::metadata(&env_cache_file) {
                if cave_meta.modified().unwrap() > cache_meta.modified().unwrap() {
                    cache_valid = false;
                }
            }
        }

        if cache_valid {
            if let Ok(content) = std::fs::read_to_string(&env_cache_file) {
                if let Ok(env_vars) = serde_json::from_str::<HashMap<String, String>>(&content) {
                    log::info!("[{}] using cached environment", cave.name);
                    return Ok(env_vars);
                }
            }
        }
    }

    log::info!("[{}] building (var: {:?})", cave.name, variant);

    let env_vars = crate::commands::package::build::build_packages(
        config,
        &settings.packages,
        &settings.options,
        &pilocal_dir,
    )?;

    // Cache the environment variables
    if let Ok(content) = serde_json::to_string_pretty(&env_vars) {
        let _ = std::fs::write(&env_cache_file, content);
    }

    log::info!("[{}] build success", cave.name);
    Ok(env_vars)
}
