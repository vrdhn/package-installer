use crate::models::config::Config;
use crate::models::cave::Cave;
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use crate::services::downloader::Downloader;
use crate::services::unarchiver::Unarchiver;
use crate::services::cache::{BuildCache, StepResult};
use crate::models::version_entry::{InstallStep, Export, VersionEntry};
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use anyhow::{Context, Result};
use rayon::prelude::*;
use log::error;
use walkdir::WalkDir;
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};
use chrono;

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

pub fn execute_build(config: &Config, cave: &Cave, variant: Option<&str>) -> Result<std::collections::HashMap<String, String>> {
    let settings = cave.get_effective_settings(variant)
        .context("Failed to get effective cave settings")?;

    log::info!("[{}] building (var: {:?})", cave.name, variant);

    let repo_config = Repositories::get_all(config);
    let build_cache = BuildCache::new(config.cache_dir.clone());

    let results: Vec<Result<(String, std::collections::HashMap<String, String>, Vec<(String, PathBuf, Vec<Export>)>)>> = settings.packages
        .par_iter()
        .map(|query| {
            let selector = PackageSelector::parse(query)
                .ok_or_else(|| anyhow::anyhow!("Invalid selector: {}", query))?;

            let (full_name, version, repo_name) = resolve::resolve_query(config, repo_config, &selector)
                .ok_or_else(|| anyhow::anyhow!("Package not found: {}", query))?;

            let pkg_ctx = format!("{}:{}={}", repo_name, full_name, version.version);
            
            execute_pipeline(config, &build_cache, cave, variant, &pkg_ctx, &version)
        })
        .collect();

    let mut all_env = std::collections::HashMap::new();
    let mut exports_to_apply = Vec::new();

    for res in results {
        match res {
            Ok((_pkg_ctx, env, exports)) => {
                for (k, v) in env {
                    all_env.insert(k, v);
                }
                exports_to_apply.extend(exports);
            }
            Err(e) => {
                error!("build failed: {}", e);
                return Err(e);
            }
        }
    }

    let pilocal_dir = config.pilocal_path(&cave.name, variant);
    fs::create_dir_all(&pilocal_dir).context("Failed to create .pilocal directory")?;

    for (pkg_ctx, source_root, exports) in exports_to_apply {
        for export in exports {
            match export {
                Export::Link { src, dest } => {
                    apply_filemap_entry(&pkg_ctx, &source_root, &pilocal_dir, &src, &dest)?;
                }
                Export::Path(rel_path) => {
                    let full_path = pilocal_dir.join(&rel_path);
                    fs::create_dir_all(full_path).ok();
                }
                Export::Env { key, val } => {
                    all_env.insert(key, val);
                }
            }
        }
    }
    
    log::info!("[{}] build success", cave.name);
    Ok(all_env)
}

fn execute_pipeline(
    config: &Config,
    build_cache: &BuildCache,
    cave: &Cave,
    variant: Option<&str>,
    pkg_ctx: &str,
    version: &VersionEntry
) -> Result<(String, std::collections::HashMap<String, String>, Vec<(String, PathBuf, Vec<Export>)>)> {
    let mut current_path: Option<PathBuf> = None;
    let mut env = std::collections::HashMap::new();

    for (i, step) in version.pipeline.iter().enumerate() {
        let step_hash = hash_step(step);
        
        if let Some(cached) = build_cache.get_step_result(&version.pkgname, &version.version, i, &step_hash) {
            log::debug!("[{}] step {} cache hit", pkg_ctx, i);
            current_path = cached.output_path;
            continue;
        }

        log::info!("[{}] executing step {}: {:?}", pkg_ctx, i, step);
        let result_path = match step {
            InstallStep::Fetch { url, checksum, filename } => {
                let fname = filename.clone().unwrap_or_else(|| url.split('/').last().unwrap_or("download").to_string());
                let dest = config.download_dir.join(fname);
                Downloader::download_to_file(url, &dest, checksum.as_deref())?;
                dest
            }
            InstallStep::Extract { format: _ } => {
                let src = current_path.as_ref().context("Extract step requires a previous Fetch step")?;
                let pkg_dir_name = format!("{}-{}-extracted", sanitize_name(&version.pkgname), sanitize_name(&version.version));
                let dest = config.packages_dir.join(pkg_dir_name);
                Unarchiver::unarchive(src, &dest)?;
                dest
            }
            InstallStep::Run { command, cwd } => {
                let base_dir = match cwd {
                    Some(c) => current_path.as_ref().context("Run with relative cwd requires previous step")?.join(c),
                    None => current_path.clone().context("Run requires previous step to define working directory")?,
                };
                
                let mut b = crate::commands::cave::run::prepare_sandbox(config, cave, variant, env.clone(), true)?;
                b.set_cwd(&base_dir);
                b.set_command("/bin/bash", &[String::from("-c"), command.clone()]);
                b.spawn().with_context(|| format!("Failed to execute pipeline command: {}", command))?;
                
                base_dir
            }
        };

        build_cache.update_step_result(&version.pkgname, &version.version, i, StepResult {
            step_hash,
            timestamp: chrono::Utc::now().to_rfc3339(),
            output_path: Some(result_path.clone()),
            status: "Success".to_string(),
        })?;
        
        current_path = Some(result_path);
    }

    let source_root = current_path.context("Pipeline must produce a source root")?;
    
    // Process static exports (Env) immediately
    for export in &version.exports {
        if let Export::Env { key, val } = export {
             env.insert(key.clone(), val.clone());
        }
    }

    Ok((pkg_ctx.to_string(), env, vec![(pkg_ctx.to_string(), source_root, version.exports.clone())]))
}

fn hash_step(step: &InstallStep) -> String {
    let mut hasher = DefaultHasher::new();
    match step {
        InstallStep::Fetch { url, checksum, filename } => {
            "fetch".hash(&mut hasher);
            url.hash(&mut hasher);
            checksum.hash(&mut hasher);
            filename.hash(&mut hasher);
        }
        InstallStep::Extract { format } => {
            "extract".hash(&mut hasher);
            format.hash(&mut hasher);
        }
        InstallStep::Run { command, cwd } => {
            "run".hash(&mut hasher);
            command.hash(&mut hasher);
            cwd.hash(&mut hasher);
        }
    }
    format!("{:x}", hasher.finish())
}

fn apply_filemap_entry(pkg_ctx: &str, pkg_dir: &Path, pilocal_dir: &Path, src_pattern: &str, dest_rel: &str) -> Result<()> {
    if src_pattern.contains('*') {
        let base_pattern = src_pattern.strip_suffix("*").unwrap_or(src_pattern);
        let search_path = pkg_dir.join(base_pattern);
        if !search_path.exists() {
            return Err(anyhow::anyhow!("[{}] source pattern missing: {}", pkg_ctx, search_path.display()));
        }

        let mut matched = false;
        if search_path.is_dir() {
            for entry in WalkDir::new(&search_path).max_depth(1).into_iter().filter_map(|e| e.ok()) {
                if entry.path() == search_path { continue; }
                let file_name = entry.file_name();
                let target_dest = pilocal_dir.join(dest_rel).join(file_name);
                create_symlink(entry.path(), &target_dest)?;
                matched = true;
            }
        }
        if !matched {
            return Err(anyhow::anyhow!("[{}] pattern '{}' no match in {}", pkg_ctx, src_pattern, pkg_dir.display()));
        }
    } else {
        let src_path = pkg_dir.join(src_pattern);
        let dest_path = pilocal_dir.join(dest_rel);
        if !src_path.exists() {
            return Err(anyhow::anyhow!("[{}] source missing: {}", pkg_ctx, src_path.display()));
        }
        let final_dest = if dest_rel.ends_with('/') || dest_path.is_dir() {
            let file_name = src_path.file_name().ok_or_else(|| anyhow::anyhow!("Invalid source filename"))?;
            dest_path.join(file_name)
        } else {
            dest_path
        };
        create_symlink(&src_path, &final_dest)?;
    }
    Ok(())
}

fn create_symlink(src: &Path, dest: &Path) -> Result<()> {
    if let Some(parent) = dest.parent() {
        fs::create_dir_all(parent)?;
    }
    if dest.exists() || dest.is_symlink() {
        let metadata = fs::symlink_metadata(dest)?;
        if metadata.is_dir() && !metadata.is_symlink() {
            fs::remove_dir_all(dest)?;
        } else {
            fs::remove_file(dest)?;
        }
    }
    #[cfg(unix)]
    {
        use std::os::unix::fs::symlink;
        symlink(src, dest)?;
    }
    Ok(())
}

fn sanitize_name(name: &str) -> String {
    name.replace(['/', '\\', ' ', ':'], "_")
}
