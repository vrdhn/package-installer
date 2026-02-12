use crate::models::config::Config;
use crate::models::cave::Cave;
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use std::env;
use rayon::prelude::*;
use comfy_table::presets::NOTHING;
use comfy_table::Table;

pub fn run(config: &Config, variant: Option<String>) {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (_path, cave) = match Cave::find_in_ancestry(&current_dir) {
        Some(res) => res,
        None => {
            println!("No cave found.");
            return;
        }
    };

    let settings = match cave.get_effective_settings(variant.as_deref()) {
        Ok(s) => s,
        Err(e) => {
            println!("Error: {}", e);
            return;
        }
    };

    println!("Resolving packages for cave {} (variant {:?})...", cave.name, variant);

    let repo_config = Repositories::get_all(config);

    let results: Vec<(String, String, String)> = settings.packages
        .par_iter()
        .map(|query| {
            let selector = match PackageSelector::parse(query) {
                Some(s) => s,
                None => return (query.clone(), "Invalid selector".to_string(), "-".to_string()),
            };

            match resolve::resolve_query(config, repo_config, &selector) {
                Some((full_name, version)) => (query.clone(), full_name, version.release_date),
                None => (query.clone(), "Not found".to_string(), "-".to_string()),
            }
        })
        .collect();

    let mut table = Table::new();
    table.load_preset(NOTHING);
    table.set_header(vec!["Query", "Resolved Full Name", "Release Date"]);
    for (query, full_name, date) in results {
        table.add_row(vec![query, full_name, date]);
    }
    println!("{table}");
}
