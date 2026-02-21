use crate::models::config::Config;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::commands::package::resolve;
use comfy_table::presets::UTF8_FULL;
use comfy_table::{Cell, Color, Table};

pub fn run(config: &Config, selector_str: &str) {
    let selector = match PackageSelector::parse(selector_str) {
        Some(s) => s,
        None => {
            log::error!("invalid selector: {}", selector_str);
            return;
        }
    };

    let repo_config = Repositories::get_all(config);
    let resolved = resolve::resolve_query(config, repo_config, &selector);

    match resolved {
        Some((full_name, version, repo_name)) => {
            print_package_info(&full_name, &version, &repo_name);
        }
        None => {
            log::error!("package not found: {}", selector_str);
        }
    }
}

fn print_package_info(full_name: &str, v: &crate::models::version_entry::VersionEntry, repo_name: &str) {
    let mut table = Table::new();
    table.load_preset(UTF8_FULL);
    table.set_header(vec![
        Cell::new("Property").fg(Color::Yellow),
        Cell::new("Value").fg(Color::Yellow),
    ]);

    table.add_row(vec!["Package", full_name]);
    table.add_row(vec!["Repository", repo_name]);
    table.add_row(vec!["Version", &v.version]);
    if !v.stream.is_empty() {
        table.add_row(vec!["Stream", &v.stream]);
    }
    table.add_row(vec!["Release Date", &v.release_date]);
    table.add_row(vec!["Release Type", &v.release_type]);

    println!("{}", table);

    if !v.pipeline.is_empty() {
        println!("\nPipeline Steps:");
        let mut pipe_table = Table::new();
        pipe_table.load_preset(UTF8_FULL);
        pipe_table.set_header(vec!["#", "Type", "Details"]);
        for (i, step) in v.pipeline.iter().enumerate() {
            let (typ, details) = match step {
                crate::models::version_entry::InstallStep::Fetch { url, .. } => ("Fetch", url.clone()),
                crate::models::version_entry::InstallStep::Extract { .. } => ("Extract", "-".to_string()),
                crate::models::version_entry::InstallStep::Run { command, .. } => ("Run", command.clone()),
            };
            pipe_table.add_row(vec![&i.to_string(), typ, &details]);
        }
        println!("{}", pipe_table);
    }

    if !v.exports.is_empty() {
        println!("\nExports:");
        let mut exp_table = Table::new();
        exp_table.load_preset(UTF8_FULL);
        exp_table.set_header(vec!["Type", "Source", "Destination/Value"]);
        for export in &v.exports {
            let (typ, src, dest) = match export {
                crate::models::version_entry::Export::Link { src, dest } => ("Link", src.clone(), dest.clone()),
                crate::models::version_entry::Export::Env { key, val } => ("Env", key.clone(), val.clone()),
                crate::models::version_entry::Export::Path(p) => ("Path", p.clone(), "-".to_string()),
            };
            exp_table.add_row(vec![typ, &src, &dest]);
        }
        println!("{}", exp_table);
    }
}
