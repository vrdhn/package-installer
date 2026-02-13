use crate::models::config::Config;
use crate::models::cave::{Cave, CaveSettings};
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use std::env;

pub fn run(config: &Config, arg1: String, arg2: Option<String>) {
    let (variant, query) = if arg1.starts_with(':') {
        if let Some(q) = arg2 {
            (Some(arg1), q)
        } else {
            log::error!("missing query after variant");
            return;
        }
    } else {
        (None, arg1)
    };

    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (path, mut cave) = match Cave::find_in_ancestry(&current_dir) {
        Some(res) => res,
        None => {
            log::error!("no cave found");
            return;
        }
    };

    // Parse query to ensure it's valid
    if PackageSelector::parse(&query).is_none() {
        log::error!("invalid query: {}", query);
        return;
    }

    // Resolve the package
    let repo_config = Repositories::get_all(config);
    let selector = PackageSelector::parse(&query).unwrap();
    
    log::info!("[{}] resolving", query);
    if let Some((full_name, version, repo_name)) = resolve::resolve_query(config, repo_config, &selector) {
        log::info!("[{}/{}] resolved: {} ({})", repo_name, full_name, version.version, version.release_type);
    } else {
        log::warn!("[{}] could not resolve, adding anyway", query);
    }
    
    let settings = if let Some(ref v_name) = variant {
        let v_name = v_name.strip_prefix(':').unwrap_or(v_name);
        cave.variants.entry(v_name.to_string()).or_insert_with(CaveSettings::default)
    } else {
        &mut cave.settings
    };

    if !settings.packages.contains(&query) {
        settings.packages.push(query.clone());
    }

    cave.save(&path).expect("Failed to save cave file");
    log::info!("[{}] added {} to {}", cave.name, query, variant.map(|v| format!("var {}", v)).unwrap_or_else(|| "default".to_string()));
}
