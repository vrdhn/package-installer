use crate::models::config::Config;
use comfy_table::presets::NOTHING;
use comfy_table::Table;
use std::fs;
use std::path::Path;
use walkdir::WalkDir;

pub fn run(config: &Config) {
    let mut table = Table::new();
    table.load_preset(NOTHING);
    table.set_header(vec!["Directory", "Path", "Size"]);

    add_row(&mut table, "Config", &config.config_dir);
    add_row(&mut table, "Cache", &config.cache_dir);
    add_row(&mut table, "State", &config.state_dir);

    println!("{table}");
}

fn add_row(table: &mut Table, name: &str, path: &Path) {
    let size = if path.exists() {
        calculate_dir_size(path)
    } else {
        0
    };

    table.add_row(vec![
        name.to_string(),
        path.to_string_lossy().to_string(),
        format_size(size),
    ]);
}

fn calculate_dir_size(path: &Path) -> u64 {
    WalkDir::new(path)
        .into_iter()
        .filter_map(|entry| entry.ok())
        .filter_map(|entry| fs::metadata(entry.path()).ok())
        .filter(|metadata| metadata.is_file())
        .map(|metadata| metadata.len())
        .sum()
}

fn format_size(size: u64) -> String {
    const KB: u64 = 1024;
    const MB: u64 = KB * 1024;
    const GB: u64 = MB * 1024;

    if size >= GB {
        format!("{:.2} GB", size as f64 / GB as f64)
    } else if size >= MB {
        format!("{:.2} MB", size as f64 / MB as f64)
    } else if size >= KB {
        format!("{:.2} KB", size as f64 / KB as f64)
    } else {
        format!("{} B", size)
    }
}
