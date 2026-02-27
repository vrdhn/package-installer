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

    // 1. Resolve all requested packages and their dependencies recursively
    let mut resolved_packages = std::collections::HashMap::new();
    let mut to_resolve = std::collections::VecDeque::from(settings.packages.clone());

    while let Some(query) = to_resolve.pop_front() {
        if resolved_packages.contains_key(&query) {
            continue;
        }

        let selector = PackageSelector::parse(&query)
            .ok_or_else(|| anyhow::anyhow!("Invalid selector: {}", query))?;

        let (_full_qualified_name, version, repo_name) = resolve::resolve_query(config, repo_config, &selector)
            .ok_or_else(|| anyhow::anyhow!("Package not found: {}", query))?;

        let dynamic_version = re_evaluate_version(config, repo_config, &repo_name, &version, &selector, &settings.options)?;
        
        // Add its dependencies to the resolution queue
        for dep in &dynamic_version.build_dependencies {
             if !resolved_packages.contains_key(&dep.name) {
                 to_resolve.push_back(dep.name.clone());
             }
        }

        resolved_packages.insert(query.clone(), (dynamic_version, repo_name));
    }

    // 2. Simple topological sort (using DFS to find dependencies first)
    let mut sorted_packages = Vec::new();
    let mut visited = std::collections::HashSet::new();
    let mut temp_visited = std::collections::HashSet::new();

    fn topo_sort(
        query: &str,
        resolved_packages: &std::collections::HashMap<String, (VersionEntry, String)>,
        visited: &mut std::collections::HashSet<String>,
        temp_visited: &mut std::collections::HashSet<String>,
        sorted_packages: &mut Vec<String>,
    ) -> Result<()> {
        if temp_visited.contains(query) {
            anyhow::bail!("Circular dependency detected involving: {}", query);
        }
        if !visited.contains(query) {
            temp_visited.insert(query.to_string());
            if let Some((version, _)) = resolved_packages.get(query) {
                for dep in &version.build_dependencies {
                    topo_sort(&dep.name, resolved_packages, visited, temp_visited, sorted_packages)?;
                }
            }
            temp_visited.remove(query);
            visited.insert(query.to_string());
            sorted_packages.push(query.to_string());
        }
        Ok(())
    }

    // Sort all discovered packages (including transitive dependencies)
    let all_queries: Vec<String> = resolved_packages.keys().cloned().collect();
    for query in &all_queries {
        topo_sort(query, &resolved_packages, &mut visited, &mut temp_visited, &mut sorted_packages)?;
    }

    // 3. Execute pipelines in sorted order
    let mut all_env = std::collections::HashMap::new();
    let pilocal_dir = config.pilocal_path(&cave.name, variant);
    fs::create_dir_all(&pilocal_dir).context("Failed to create .pilocal directory")?;

    for query in sorted_packages {
        let (dynamic_version, repo_name) = resolved_packages.get(&query).unwrap();
        let pkg_ctx = format!("{}:{}={}", repo_name, dynamic_version.pkgname, dynamic_version.version);
        
        let (_ctx, env, exports) = execute_pipeline(config, &build_cache, cave, variant, &pkg_ctx, dynamic_version, repo_name)?;
        
        for (k, v) in env {
            all_env.insert(k, v);
        }

        // Apply exports immediately so they are available for the next packages
        for (pkg_ctx, source_root, pkg_exports) in exports {
            for export in pkg_exports {
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
    }
    
    log::info!("[{}] build success", cave.name);
    Ok(all_env)
}

fn re_evaluate_version(
    config: &Config,
    repo_config: &Repositories,
    repo_name: &str,
    version: &VersionEntry,
    selector: &PackageSelector,
    all_options: &std::collections::HashMap<String, std::collections::HashMap<String, serde_json::Value>>
) -> Result<VersionEntry> {
    if let Some(res) = re_evaluate_version_internal(config, repo_config, repo_name, version, selector, all_options, false)? {
        return Ok(res);
    }

    // Retry with forced sync if not found
    if !config.force {
        log::debug!("[{}] not found in repository cache during re-evaluation, attempting sync", version.pkgname);
        if let Some(res) = re_evaluate_version_internal(config, repo_config, repo_name, version, selector, all_options, true)? {
            return Ok(res);
        }
    }

    anyhow::bail!("Package entry '{}' not found in repository '{}' during re-evaluation", version.pkgname, repo_name);
}

fn re_evaluate_version_internal(
    config: &Config,
    repo_config: &Repositories,
    repo_name: &str,
    version: &VersionEntry,
    selector: &PackageSelector,
    all_options: &std::collections::HashMap<String, std::collections::HashMap<String, serde_json::Value>>,
    force: bool,
) -> Result<Option<VersionEntry>> {
    let repo = repo_config.repositories.iter().find(|r| r.name == repo_name)
        .context(format!("Repository '{}' not found during re-evaluation", repo_name))?;
    
    let pkg_list = crate::models::package_entry::PackageList::get_for_repo(config, repo, force)
        .context(format!("Package list for repository '{}' not found", repo_name))?;
    
    // Look for direct package
    let pkg_entry = pkg_list.package_map.get(&version.pkgname);
    
    log::debug!("re-evaluating {} (version {})", version.pkgname, version.version);

    // If not found, check if it's a manager package
    let prefix = selector.prefix.as_ref();
    let manager_entry = if pkg_entry.is_none() {
        if let Some(prefix) = prefix {
            pkg_list.manager_map.get(prefix)
        } else if version.pkgname.contains(':') {
             let p = version.pkgname.split(':').next().unwrap();
             pkg_list.manager_map.get(p)
        } else {
            // Check if any manager matches the pkgname (legacy)
            pkg_list.manager_map.get(&version.pkgname)
        }
    } else {
        None
    };

    let star_path = if let Some(pkg) = pkg_entry {
        Path::new(&repo.path).join(&pkg.filename)
    } else if let Some(mgr) = manager_entry {
        Path::new(&repo.path).join(&mgr.filename)
    } else {
        return Ok(None);
    };

    let function_name = if let Some(pkg) = pkg_entry {
        &pkg.function_name
    } else {
        &manager_entry.unwrap().function_name
    };

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

    let is_manager = manager_entry.is_some();
    let dynamic_versions = if is_manager {
        let package_name = if version.pkgname.contains(':') {
            version.pkgname.split(':').nth(1).unwrap()
        } else {
            &version.pkgname
        };

        crate::starlark::runtime::execute_manager_function(
            &star_path,
            function_name,
            prefix.map(|s| s.as_str()).unwrap_or_else(|| version.pkgname.split(':').next().unwrap()),
            package_name,
            config.state.clone(),
            Some(options),
        )?
    } else {
        crate::starlark::runtime::execute_function(
            &star_path,
            function_name,
            &version.pkgname,
            config.state.clone(),
            Some(options),
        )?
    };

    Ok(dynamic_versions.into_iter().find(|v| v.version == version.version))
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
        // Resolve dependency using its selector string
        let selector = match PackageSelector::parse(&dep.name) {
            Some(s) => s,
            None => {
                if !dep.optional {
                    anyhow::bail!("[{}] invalid dependency selector: {}", pkg_ctx, dep.name);
                }
                continue;
            }
        };

        if let Some((_, dep_version, dep_repo)) = resolve::resolve_query(config, repo_config, &selector) {
            // We need to re-evaluate the dependency to get its context-aware installation path
            let settings = cave.get_effective_settings(variant)?;
            let dynamic_dep = re_evaluate_version(config, repo_config, &dep_repo, &dep_version, &selector, &settings.options)?;
            
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
                            let parent_buf = parent.to_path_buf();
                            if !dependency_dirs.contains(&parent_buf) {
                                dependency_dirs.push(parent_buf);
                            }
                        }
                    }
                }
            }
        } else if !dep.optional {
            anyhow::bail!("[{}] missing required build dependency: {}", pkg_ctx, dep.name);
        }
    }

    let mut recomputed = false;
    for (i, step) in version.pipeline.iter().enumerate() {
        // Resolve @PACKAGES_DIR placeholder in Run commands before hashing
        let mut resolved_step = step.clone();
        if let InstallStep::Run { ref mut command, .. } = resolved_step {
            *command = command.replace("@PACKAGES_DIR", config.packages_dir.to_str().unwrap());
        }

        let step_hash = hash_to_string(&resolved_step);
        
        if !config.force && !recomputed {
            if let Some(cached) = build_cache.get_step_result(&version.pkgname, &version.version.to_string(), i, &step_hash) {
                log::debug!("[{}] step {} cache hit", pkg_ctx, i);
                current_path = cached.output_path;
                continue;
            }
        }

        recomputed = true;
        log::info!("[{}] executing step {}: {:?}", pkg_ctx, i, resolved_step);
        let result_path = execute_step(config, cave, variant, &resolved_step, &current_path, &env, &version.pkgname, &version.version.to_string(), dependency_dirs.clone())?;

        let step_name = match resolved_step {
            InstallStep::Fetch { name, .. } => name.clone(),
            InstallStep::Extract { name, .. } => name.clone(),
            InstallStep::Run { name, .. } => name.clone(),
        };

        build_cache.update_step_result(&version.pkgname, &version.version.to_string(), i, StepResult {
            name: step_name,
            step_hash,
            timestamp: chrono::Utc::now().to_rfc3339(),
            output_path: Some(result_path.clone()),
            status: "Success".to_string(),
        })?;
        
        current_path = Some(result_path);
    }

    let source_root = current_path.unwrap_or_else(|| {
        let pkg_dir_name = format!("{}-{}", sanitize_name(&version.pkgname), sanitize_name(&version.version.to_string()));
        config.packages_dir.join(pkg_dir_name)
    });
    
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
            let pkg_dir_name = format!("{}-{}", sanitize_name(pkgname), sanitize_name(version));
            let default_base = config.packages_dir.join(pkg_dir_name);
            
            let base_dir = match cwd {
                Some(c) => {
                    let base = current_path.as_ref().unwrap_or(&default_base);
                    base.join(c)
                }
                None => current_path.clone().unwrap_or(default_base),
            };

            // Ensure directory exists
            fs::create_dir_all(&base_dir).ok();
            
            let mut b = crate::commands::cave::run::prepare_sandbox(config, cave, variant, env.clone(), true, dependency_dirs)?;
            b.set_cwd(&base_dir);
            b.set_command("/bin/bash", &[String::from("-c"), command.clone()]);
            b.spawn().with_context(|| format!("Failed to execute pipeline command: {}", command))?;
            
            Ok(base_dir)
        }
    }
}
