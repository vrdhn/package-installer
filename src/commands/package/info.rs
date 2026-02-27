use crate::models::config::Config;
use crate::models::repository::Repositories;
use crate::models::selector::PackageSelector;
use crate::models::version_entry::VersionEntry;
use crate::commands::package::resolve;
use comfy_table::presets::UTF8_FULL;
use comfy_table::{Cell, Color, Table};

/// Options for re-evaluating a package version.
struct ReEvalOptions<'a> {
    config: &'a Config,
    repo_config: &'a Repositories,
    repo_name: &'a str,
    version: &'a VersionEntry,
    selector: &'a PackageSelector,
}

pub fn run(config: &Config, selector_str: &str) {
    let selector = match PackageSelector::parse(selector_str) {
        Some(s) => s,
        None => {
            log::error!("invalid selector: {}", selector_str);
            return;
        }
    };

    let repo_config = Repositories::get_all(config);
    let resolved = resolve::resolve_query(config, &repo_config, &selector);

    match resolved {
        Some((full_name, version, repo_name)) => {
            let opts = ReEvalOptions {
                config, repo_config: &repo_config, repo_name: &repo_name,
                version: &version, selector: &selector,
            };
            let dynamic_version = re_evaluate_version(opts);
            print_package_info(&full_name, &dynamic_version.unwrap_or(version), &repo_name);
        }
        None => log::error!("package not found: {}", selector_str),
    }
}

fn re_evaluate_version(opts: ReEvalOptions) -> Option<VersionEntry> {
    let repo = opts.repo_config.repositories.iter().find(|r| r.name == opts.repo_name)?;
    let pkg_list = crate::models::package_entry::PackageList::get_for_repo(opts.config, repo, false)?;
    
    let (star_file, func, arg) = find_entry_details(&pkg_list, opts.version, opts.selector)?;
    let star_path = std::path::Path::new(&repo.path).join(&star_file);
    
    let exec_opts = crate::starlark::runtime::ExecutionOptions {
        path: &star_path, function_name: &func, config: opts.config, options: None,
    };

    let dynamic_versions = if opts.version.pkgname.contains(':') {
        let mgr_name = opts.version.pkgname.split(':').next()?;
        crate::starlark::runtime::execute_manager_function(exec_opts, mgr_name, &arg).ok()?
    } else {
        crate::starlark::runtime::execute_function(exec_opts, &arg).ok()?
    };

    dynamic_versions.into_iter().find(|v| v.version == opts.version.version)
}

fn find_entry_details(
    pkg_list: &crate::models::package_entry::PackageList,
    version: &VersionEntry,
    selector: &PackageSelector
) -> Option<(String, String, String)> {
    if let Some(pkg) = pkg_list.packages.get(&version.pkgname) {
        return Some((pkg.filename.clone(), pkg.function_name.clone(), pkg.name.clone()));
    }
    
    if let Some(prefix) = selector.prefix.as_ref() {
        if let Some(mgr) = pkg_list.managers.get(prefix) {
            let inner = if version.pkgname.contains(':') {
                version.pkgname.split(':').nth(1).unwrap().to_string()
            } else {
                version.pkgname.clone()
            };
            return Some((mgr.filename.clone(), mgr.function_name.clone(), inner));
        }
    }

    if version.pkgname.contains(':') {
        let mgr_name = version.pkgname.split(':').next()?;
        if let Some(mgr) = pkg_list.managers.get(mgr_name) {
            let inner = version.pkgname.split(':').nth(1)?;
            return Some((mgr.filename.clone(), mgr.function_name.clone(), inner.to_string()));
        }
    }
    None
}

fn print_package_info(full_name: &str, v: &VersionEntry, repo_name: &str) {
    print_base_info(full_name, v, repo_name);
    
    if !v.build_dependencies.is_empty() {
        print_dependencies(&v.build_dependencies);
    }
    if !v.pipeline.is_empty() {
        print_pipeline(&v.pipeline);
    }
    if !v.exports.is_empty() {
        print_exports(&v.exports);
    }
}

fn print_base_info(full_name: &str, v: &VersionEntry, repo_name: &str) {
    let mut table = Table::new();
    table.load_preset(UTF8_FULL);
    table.set_header(vec![
        Cell::new("Property").fg(Color::Yellow),
        Cell::new("Value").fg(Color::Yellow),
    ]);

    table.add_row(vec!["Package", full_name]);
    table.add_row(vec!["Repository", repo_name]);
    table.add_row(vec!["Version", &v.version.to_string()]);
    if !v.stream.is_empty() { table.add_row(vec!["Stream", &v.stream]); }
    table.add_row(vec!["Release Date", &v.release_date]);
    table.add_row(vec!["Release Type", &v.release_type.to_string()]);
    println!("{}", table);
}

fn print_dependencies(deps: &[crate::models::version_entry::Dependency]) {
    println!("\nBuild Dependencies:");
    let mut table = Table::new();
    table.load_preset(UTF8_FULL);
    table.set_header(vec!["Package", "Optional"]);
    for dep in deps {
        table.add_row(vec![&dep.name, &dep.optional.to_string()]);
    }
    println!("{}", table);
}

fn print_pipeline(steps: &[crate::models::version_entry::InstallStep]) {
    println!("\nPipeline Steps:");
    let mut table = Table::new();
    table.load_preset(UTF8_FULL);
    table.set_header(vec!["#", "Name", "Type", "Details"]);
    for (i, step) in steps.iter().enumerate() {
        let (typ, details, name) = match step {
            crate::models::version_entry::InstallStep::Fetch { url, name, .. } => ("Fetch", url.clone(), name.as_deref().unwrap_or("-")),
            crate::models::version_entry::InstallStep::Extract { name, .. } => ("Extract", "-".to_string(), name.as_deref().unwrap_or("-")),
            crate::models::version_entry::InstallStep::Run { command, name, .. } => ("Run", command.clone(), name.as_deref().unwrap_or("-")),
        };
        table.add_row(vec![&i.to_string(), name, typ, &details]);
    }
    println!("{}", table);
}

fn print_exports(exports: &[crate::models::version_entry::Export]) {
    println!("\nExports:");
    let mut table = Table::new();
    table.load_preset(UTF8_FULL);
    table.set_header(vec!["Type", "Source", "Destination/Value"]);
    for export in exports {
        let (typ, src, dest) = match export {
            crate::models::version_entry::Export::Link { src, dest } => ("Link", src.clone(), dest.clone()),
            crate::models::version_entry::Export::Env { key, val } => ("Env", key.clone(), val.clone()),
            crate::models::version_entry::Export::Path(p) => ("Path", p.clone(), "-".to_string()),
        };
        table.add_row(vec![typ, &src, &dest]);
    }
    println!("{}", table);
}
