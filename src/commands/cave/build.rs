use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;
use std::collections::HashMap;
use anyhow::{Context, Result};

pub fn run(config: &Config, variant: Option<String>) {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (_path, cave) = match Cave::find_in_ancestry(&current_dir) {
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
    log::info!("[{}] building (var: {:?})", cave.name, variant);

    let pilocal_dir = config.pilocal_path(&cave.name, variant);
    
    let env_vars = crate::commands::package::build::build_packages(
        config,
        &settings.packages,
        &settings.options,
        &pilocal_dir,
    )?;

    log::info!("[{}] build success", cave.name);
    Ok(env_vars)
}
