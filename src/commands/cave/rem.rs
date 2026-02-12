use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;

pub fn run(_config: &Config, arg1: String, arg2: Option<String>) {
    let (variant, query) = if arg1.starts_with(':') {
        if let Some(q) = arg2 {
            (Some(arg1), q)
        } else {
            println!("Error: Missing package query after variant.");
            return;
        }
    } else {
        (None, arg1)
    };

    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (path, mut cave) = match Cave::find_in_ancestry(&current_dir) {
        Some(res) => res,
        None => {
            println!("No cave found.");
            return;
        }
    };

    let settings = if let Some(ref v_name) = variant {
        let v_name = v_name.strip_prefix(':').unwrap_or(v_name);
        match cave.variants.get_mut(v_name) {
            Some(s) => s,
            None => {
                println!("Variant '{}' not found in cave.", v_name);
                return;
            }
        }
    } else {
        &mut cave.settings
    };

    let original_len = settings.packages.len();
    settings.packages.retain(|p| p != &query);

    if settings.packages.len() < original_len {
        cave.save(&path).expect("Failed to save cave file");
        println!("Removed {} from cave {}{}", query, cave.name, variant.map(|v| format!(" (variant {})", v)).unwrap_or_default());
    } else {
        println!("Package {} not found in cave {}{}", query, cave.name, variant.map(|v| format!(" (variant {})", v)).unwrap_or_default());
    }
}
