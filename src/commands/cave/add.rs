use crate::models::config::Config;
use crate::models::cave::{Cave, CaveSettings};
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use std::env;

pub fn run(config: &Config, args: Vec<String>) {
    if args.is_empty() {
        return;
    }

    let (variant, queries) = if args[0].starts_with(':') {
        (Some(args[0].clone()), args[1..].to_vec())
    } else {
        (None, args)
    };

    if queries.is_empty() {
        log::error!("missing package query");
        return;
    }

    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (path, mut cave) = match Cave::find_in_ancestry(&current_dir) {
        Some(res) => res,
        None => {
            log::error!("no cave found");
            return;
        }
    };

    let repo_config = Repositories::get_all(config);
    
    for query in queries {
        // Parse query to ensure it's valid
        if PackageSelector::parse(&query).is_none() {
            log::error!("invalid query: {}", query);
            continue;
        }

        // Resolve the package
        let selector = PackageSelector::parse(&query).unwrap();
        
        log::info!("[{}] resolving", query);
        if let Some((full_name, version, repo_name)) = resolve::resolve_query(config, repo_config, &selector) {
            log::info!("[{}/{}] resolved: {} ({})", repo_name, full_name, version.version.to_string(), version.release_type.to_string());
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
        
        log::info!("[{}] added {} to {}", cave.name, query, variant.as_deref().unwrap_or("default"));
    }

    cave.save(&path).expect("Failed to save cave file");
}
