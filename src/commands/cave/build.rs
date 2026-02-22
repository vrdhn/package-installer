use crate::models::config::Config;
use crate::models::cave::Cave;
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use crate::services::downloader::Downloader;
use crate::services::unarchiver::Unarchiver;
use crate::services::cache::{BuildCache, StepResult};
use crate::models::version_entry::{InstallStep, Export, VersionEntry};
use crate::commands::cave::fs::apply_filemap_entry;
use crate::utils::fs::sanitize_name;
use crate::utils::crypto::hash_to_string;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use anyhow::{Context, Result};
use rayon::prelude::*;
use log::error;
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

            let (_full_qualified_name, version, repo_name) = resolve::resolve_query(config, repo_config, &selector)
                .ok_or_else(|| anyhow::anyhow!("Package not found: {}", query))?;

            // Re-run Starlark with Cave options to get the context-aware pipeline and exports
            let dynamic_version = re_evaluate_version(config, repo_config, &repo_name, &version, &settings.options)?;

            let pkg_ctx = format!("{}:{}={}", repo_name, dynamic_version.pkgname, dynamic_version.version);
            
            execute_pipeline(config, &build_cache, cave, variant, &pkg_ctx, &dynamic_version, &repo_name)
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
                    let src = src.replace("@PACKAGES_DIR", config.packages_dir.to_str().unwrap());
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

fn re_evaluate_version(
    config: &Config,
    repo_config: &Repositories,
    repo_name: &str,
    version: &VersionEntry,
    all_options: &std::collections::HashMap<String, std::collections::HashMap<String, serde_json::Value>>
) -> Result<VersionEntry> {
    let repo = repo_config.repositories.iter().find(|r| r.name == repo_name)
        .context("Repository not found during re-evaluation")?;
    
    let pkg_list = crate::models::package_entry::PackageList::get_for_repo(config, repo)
        .context("Package list not found")?;
    
    let pkg_entry = pkg_list.packages.iter().find(|p| p.name == version.pkgname)
        .context("Package entry not found during re-evaluation")?;

    let star_path = Path::new(&repo.path).join(&pkg_entry.filename);
    
    let mut options = std::collections::HashMap::new();
    if let Some(pkg_opts) = all_options.get(&version.pkgname) {
        for (k, v) in pkg_opts {
            options.insert(k.clone(), match v {
                serde_json::Value::String(s) => s.clone(),
                serde_json::Value::Bool(b) => b.to_string(),
                _ => v.to_string(),
            });
        }
    }

    let dynamic_versions = crate::starlark::runtime::execute_function(
        &star_path,
        &pkg_entry.function_name,
        &pkg_entry.name,
        config.state.clone(),
        Some(options),
    )?;

    dynamic_versions.into_iter().find(|v| v.version == version.version)
        .context(format!("Version {} not found after re-evaluation of {}", version.version, version.pkgname))
}

fn execute_pipeline(
    config: &Config,
    build_cache: &BuildCache,
    cave: &Cave,
    variant: Option<&str>,
    pkg_ctx: &str,
    version: &VersionEntry,
    _repo_name: &str,
) -> Result<(String, std::collections::HashMap<String, String>, Vec<(String, PathBuf, Vec<Export>)>)> {
    let mut current_path: Option<PathBuf> = None;
    let mut env = std::collections::HashMap::new();
    let repo_config = Repositories::get_all(config);

    // 1. Resolve build-time dependencies
    let mut dependency_dirs = Vec::new();
    for dep in &version.build_dependencies {
        // Resolve dependency by name (no version for now as requested)
        let selector = PackageSelector {
            recipe: None,
            prefix: None,
            package: dep.name.clone(),
            version: None,
        };

        if let Some((_, dep_version, dep_repo)) = resolve::resolve_query(config, repo_config, &selector) {
            // We need to re-evaluate the dependency to get its context-aware installation path
            let settings = cave.get_effective_settings(variant)?;
            let dynamic_dep = re_evaluate_version(config, repo_config, &dep_repo, &dep_version, &settings.options)?;
            
            // For now, we assume dependencies are already built or we just find their potential path.
            // In a more robust system, we would trigger their build first (topological sort).
            // Let's at least try to find the install dir if it exists.
            let _dep_pkg_ctx = format!("{}:{}={}", dep_repo, dynamic_dep.pkgname, dynamic_dep.version);
            
            // Heuristic: dependencies usually install to @PACKAGES_DIR/{pkg}-{version}-{flags_hash}
            // But we can extract it from their 'Run' command or 'Export::Link' if we had a more structured model.
            // For Erlang specifically, we know how it's calculated.
            // Let's use a simpler approach for now: find any Export::Link that is absolute.
            for export in &dynamic_dep.exports {
                if let Export::Link { src, .. } = export {
                    let resolved_src = src.replace("@PACKAGES_DIR", config.packages_dir.to_str().unwrap());
                    let p = Path::new(&resolved_src);
                    if p.is_absolute() {
                        // Get the directory containing bin/ or lib/
                        if let Some(parent) = p.parent() {
                            if !dependency_dirs.contains(&parent.to_path_buf()) {
                                dependency_dirs.push(parent.to_path_buf());
                            }
                        }
                    }
                }
            }
        } else if !dep.optional {
            anyhow::bail!("[{}] missing required build dependency: {}", pkg_ctx, dep.name);
        }
    }

    for (i, step) in version.pipeline.iter().enumerate() {
        // Resolve @PACKAGES_DIR placeholder in Run commands before hashing
        let mut resolved_step = step.clone();
        if let InstallStep::Run { ref mut command, .. } = resolved_step {
            *command = command.replace("@PACKAGES_DIR", config.packages_dir.to_str().unwrap());
        }

        let step_hash = hash_to_string(&resolved_step);
        
        if let Some(cached) = build_cache.get_step_result(&version.pkgname, &version.version, i, &step_hash) {
            log::debug!("[{}] step {} cache hit", pkg_ctx, i);
            current_path = cached.output_path;
            continue;
        }

        log::info!("[{}] executing step {}: {:?}", pkg_ctx, i, resolved_step);
        let result_path = execute_step(config, cave, variant, &resolved_step, &current_path, &env, &version.pkgname, &version.version, dependency_dirs.clone())?;

        let step_name = match resolved_step {
            InstallStep::Fetch { name, .. } => name.clone(),
            InstallStep::Extract { name, .. } => name.clone(),
            InstallStep::Run { name, .. } => name.clone(),
        };

        build_cache.update_step_result(&version.pkgname, &version.version, i, StepResult {
            name: step_name,
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

fn execute_step(
    config: &Config,
    cave: &Cave,
    variant: Option<&str>,
    step: &InstallStep,
    current_path: &Option<PathBuf>,
    env: &std::collections::HashMap<String, String>,
    pkgname: &str,
    version: &str,
    dependency_dirs: Vec<PathBuf>,
) -> Result<PathBuf> {
    match step {
        InstallStep::Fetch { url, checksum, filename, .. } => {
            let fname = filename.clone().unwrap_or_else(|| url.split('/').last().unwrap_or("download").to_string());
            let dest = config.download_dir.join(fname);
            Downloader::download_to_file(url, &dest, checksum.as_deref())?;
            Ok(dest)
        }
        InstallStep::Extract { .. } => {
            let src = current_path.as_ref().context("Extract step requires a previous Fetch step")?;
            let pkg_dir_name = format!("{}-{}-extracted", sanitize_name(pkgname), sanitize_name(version));
            let dest = config.packages_dir.join(pkg_dir_name);
            Unarchiver::unarchive(src, &dest)?;
            Ok(dest)
        }
        InstallStep::Run { command, cwd, .. } => {
            let base_dir = match cwd {
                Some(c) => current_path.as_ref().context("Run with relative cwd requires previous step")?.join(c),
                None => current_path.clone().context("Run requires previous step to define working directory")?,
            };
            
            let mut b = crate::commands::cave::run::prepare_sandbox(config, cave, variant, env.clone(), true, dependency_dirs)?;
            b.set_cwd(&base_dir);
            b.set_command("/bin/bash", &[String::from("-c"), command.clone()]);
            b.spawn().with_context(|| format!("Failed to execute pipeline command: {}", command))?;
            
            Ok(base_dir)
        }
    }
}
