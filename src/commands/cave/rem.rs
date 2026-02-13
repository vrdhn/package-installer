use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;

pub fn run(_config: &Config, args: Vec<String>) {
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

    let settings = if let Some(ref v_name) = variant {
        let v_name = v_name.strip_prefix(':').unwrap_or(v_name);
        match cave.variants.get_mut(v_name) {
            Some(s) => s,
            None => {
                log::error!("variant {} not found", v_name);
                return;
            }
        }
    } else {
        &mut cave.settings
    };

    for query in queries {
        let original_len = settings.packages.len();
        settings.packages.retain(|p| p != &query);

        if settings.packages.len() < original_len {
            log::info!("[{}] removed {} from {}", cave.name, query, variant.as_deref().unwrap_or("default"));
        } else {
            log::warn!("[{}] pkg {} not found in {}", cave.name, query, variant.as_deref().unwrap_or("default"));
        }
    }

    cave.save(&path).expect("Failed to save cave file");
}
