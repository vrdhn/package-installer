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
            // Re-evaluate to get pipeline/deps (skipped in cache)
            let dynamic_version = re_evaluate_version(config, repo_config, &repo_name, &version);
            print_package_info(&full_name, &dynamic_version.unwrap_or(version), &repo_name);
        }
        None => {
            log::error!("package not found: {}", selector_str);
        }
    }
}

fn re_evaluate_version(
    config: &Config,
    repo_config: &Repositories,
    repo_name: &str,
    version: &crate::models::version_entry::VersionEntry,
) -> Option<crate::models::version_entry::VersionEntry> {
    let repo = repo_config.repositories.iter().find(|r| r.name == repo_name)?;
    let pkg_list = crate::models::package_entry::PackageList::get_for_repo(config, repo)?;
    
    // Find the package or manager entry
    let (star_file, function_name, argument) = if let Some(pkg) = pkg_list.packages.iter().find(|p| p.name == version.pkgname) {
        (pkg.filename.clone(), pkg.function_name.clone(), pkg.name.clone())
    } else if let Some(mgr) = pkg_list.managers.iter().find(|m| version.pkgname.starts_with(&format!("{}:", m.name))) {
        let pkg_inner = version.pkgname.split(':').nth(1)?;
        (mgr.filename.clone(), mgr.function_name.clone(), pkg_inner.to_string())
    } else {
        return None;
    };

    let star_path = std::path::Path::new(&repo.path).join(&star_file);
    
    // For info, we use empty options
    let dynamic_versions = if version.pkgname.contains(':') {
        let mgr_name = version.pkgname.split(':').next()?;
        crate::starlark::runtime::execute_manager_function(
            &star_path,
            &function_name,
            mgr_name,
            &argument,
            config.state.clone(),
            None,
        ).ok()?
    } else {
        crate::starlark::runtime::execute_function(
            &star_path,
            &function_name,
            &argument,
            config.state.clone(),
            None,
        ).ok()?
    };

    dynamic_versions.into_iter().find(|v| v.version == version.version)
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
    table.add_row(vec!["Version", &v.version.to_string()]);
    if !v.stream.is_empty() {
        table.add_row(vec!["Stream", &v.stream]);
    }
    table.add_row(vec!["Release Date", &v.release_date]);
    table.add_row(vec!["Release Type", &v.release_type.to_string()]);
    println!("{}", table);
    
        log::debug!("package info: deps={}, pipeline={}, exports={}", v.build_dependencies.len(), v.pipeline.len(), v.exports.len());
    
        if !v.build_dependencies.is_empty() {
    
        println!("\nBuild Dependencies:");
        let mut dep_table = Table::new();
        dep_table.load_preset(UTF8_FULL);
        dep_table.set_header(vec!["Package", "Optional"]);
        for dep in &v.build_dependencies {
            dep_table.add_row(vec![&dep.name, &dep.optional.to_string()]);
        }
        println!("{}", dep_table);
    }

    if !v.pipeline.is_empty() {
        println!("\nPipeline Steps:");
        let mut pipe_table = Table::new();
        pipe_table.load_preset(UTF8_FULL);
        pipe_table.set_header(vec!["#", "Name", "Type", "Details"]);
        for (i, step) in v.pipeline.iter().enumerate() {
            let (typ, details, name) = match step {
                crate::models::version_entry::InstallStep::Fetch { url, name, .. } => ("Fetch", url.clone(), name.as_deref().unwrap_or("-")),
                crate::models::version_entry::InstallStep::Extract { name, .. } => ("Extract", "-".to_string(), name.as_deref().unwrap_or("-")),
                crate::models::version_entry::InstallStep::Run { command, name, .. } => ("Run", command.clone(), name.as_deref().unwrap_or("-")),
            };
            pipe_table.add_row(vec![&i.to_string(), name, typ, &details]);
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
