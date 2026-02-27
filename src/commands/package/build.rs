use crate::models::config::Config;
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use crate::services::downloader::Downloader;
use crate::services::unarchiver::Unarchiver;
use crate::services::cache::{BuildCache, StepResult};
use crate::models::version_entry::{InstallStep, Export, VersionEntry, QualifiedVersion};
use crate::commands::cave::fs::apply_filemap_entry;
use crate::utils::fs::sanitize_name;
use crate::utils::crypto::hash_to_string;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use anyhow::{Context, Result};
use chrono;
use std::collections::{HashMap, HashSet, VecDeque};

pub struct BuildContext<'a> {
    pub config: &'a Config,
    pub repo_config: &'a Repositories,
    pub build_cache: &'a BuildCache,
    pub all_options: &'a HashMap<String, HashMap<String, serde_json::Value>>,
    pub pilocal_dir: &'a Path,
}

pub struct StepContext<'a> {
    pub config: &'a Config,
    pub env: &'a HashMap<String, String>,
    pub dependency_dirs: Vec<PathBuf>,
    pub pkgname: &'a str,
    pub version: &'a str,
    pub pilocal_dir: &'a Path,
}

pub fn build_packages(
    config: &Config,
    packages: &[String],
    all_options: &HashMap<String, HashMap<String, serde_json::Value>>,
    pilocal_dir: &Path,
) -> Result<HashMap<String, String>> {
    let repo_config = Repositories::get_all(config);
    let build_cache = BuildCache::new(config.cache_dir.clone());

    let ctx = BuildContext {
        config,
        repo_config: &repo_config,
        build_cache: &build_cache,
        all_options,
        pilocal_dir,
    };

    let resolved_packages = resolve_dependencies(&ctx, packages)?;
    let sorted_packages = topological_sort(&resolved_packages)?;

    execute_sorted_pipelines(&ctx, sorted_packages, &resolved_packages)
}

fn resolve_dependencies(
    ctx: &BuildContext,
    initial_packages: &[String]
) -> Result<HashMap<String, (VersionEntry, String)>> {
    let mut resolved = HashMap::new();
    let mut to_resolve = VecDeque::from(initial_packages.to_vec());

    while let Some(query) = to_resolve.pop_front() {
        if resolved.contains_key(&query) { continue; }

        let selector = PackageSelector::parse(&query).ok_or_else(|| anyhow::anyhow!("Invalid selector: {}", query))?;
        let (_, version, repo_name) = resolve::resolve_query(ctx.config, ctx.repo_config, &selector)
            .ok_or_else(|| anyhow::anyhow!("Package not found: {}", query))?;

        let dynamic_version = re_evaluate_version(ctx, &repo_name, &version, &selector)?;

        for dep in &dynamic_version.build_dependencies {
            if !resolved.contains_key(&dep.name) {
                to_resolve.push_back(dep.name.clone());
            }
        }

        resolved.insert(query, (dynamic_version, repo_name));
    }
    Ok(resolved)
}

fn topological_sort(resolved_packages: &HashMap<String, (VersionEntry, String)>) -> Result<Vec<String>> {
    let mut sorted = Vec::new();
    let mut visited = HashSet::new();
    let mut temp_visited = HashSet::new();

    for query in resolved_packages.keys() {
        topo_sort_dfs(query, resolved_packages, &mut visited, &mut temp_visited, &mut sorted)?;
    }
    Ok(sorted)
}

fn topo_sort_dfs(
    query: &str,
    resolved: &HashMap<String, (VersionEntry, String)>,
    visited: &mut HashSet<String>,
    temp_visited: &mut HashSet<String>,
    sorted: &mut Vec<String>,
) -> Result<()> {
    if temp_visited.contains(query) { anyhow::bail!("Circular dependency involving: {}", query); }
    if !visited.contains(query) {
        temp_visited.insert(query.to_string());
        if let Some((version, _)) = resolved.get(query) {
            for dep in &version.build_dependencies {
                topo_sort_dfs(&dep.name, resolved, visited, temp_visited, sorted)?;
            }
        }
        temp_visited.remove(query);
        visited.insert(query.to_string());
        sorted.push(query.to_string());
    }
    Ok(())
}

fn execute_sorted_pipelines(
    ctx: &BuildContext,
    sorted_packages: Vec<String>,
    resolved_packages: &HashMap<String, (VersionEntry, String)>
) -> Result<HashMap<String, String>> {
    let mut all_env = HashMap::new();
    fs::create_dir_all(ctx.pilocal_dir).context("Failed to create .pilocal dir")?;

    for query in sorted_packages {
        let (dyn_version, repo_name) = resolved_packages.get(&query).unwrap();
        let qv = QualifiedVersion::new(repo_name, dyn_version);

        let (_, env, exports) = execute_pipeline(ctx, &qv.pkg_ctx(), dyn_version, repo_name)?;
        all_env.extend(env);

        apply_exports(ctx, exports, ctx.pilocal_dir, &mut all_env)?;
    }

    Ok(all_env)
}

fn apply_exports(
    ctx: &BuildContext,
    exports: Vec<(String, PathBuf, Vec<Export>)>,
    pilocal_dir: &Path,
    all_env: &mut HashMap<String, String>
) -> Result<()> {
    for (pkg_ctx, source_root, pkg_exports) in exports {
        for export in pkg_exports {
            match export {
                Export::Link { src, dest } => {
                    let src = ctx.config.resolve_packages_dir(&src);
                    apply_filemap_entry(crate::commands::cave::fs::FileMapOptions {
                        pkg_ctx: &pkg_ctx,
                        pkg_dir: &source_root,
                        pilocal_dir,
                        src_pattern: &src,
                        dest_rel: &dest,
                    })?;
                }
                Export::Path(rel_path) => {
                    fs::create_dir_all(pilocal_dir.join(&rel_path)).ok();
                }
                Export::Env { key, val } => {
                    all_env.insert(key, val);
                }
            }
        }
    }
    Ok(())
}

fn re_evaluate_version(
    ctx: &BuildContext,
    repo_name: &str,
    version: &VersionEntry,
    selector: &PackageSelector,
) -> Result<VersionEntry> {
    if let Some(res) = re_evaluate_version_internal(ctx, repo_name, version, selector, false)? {
        return Ok(res);
    }
    if !ctx.config.force {
        log::debug!("[{}] not found in cache, attempting sync", version.pkgname);
        if let Some(res) = re_evaluate_version_internal(ctx, repo_name, version, selector, true)? {
            return Ok(res);
        }
    }
    anyhow::bail!("Package entry '{}' not found in repo '{}'", version.pkgname, repo_name);
}

fn re_evaluate_version_internal(
    ctx: &BuildContext,
    repo_name: &str,
    version: &VersionEntry,
    selector: &PackageSelector,
    force: bool,
) -> Result<Option<VersionEntry>> {
    let repo = ctx.repo_config.repositories.iter().find(|r| r.name == repo_name)
        .context(format!("Repo '{}' not found", repo_name))?;
    let pkg_list = crate::models::package_entry::PackageList::get_for_repo(ctx.config, repo, force)
        .context(format!("Package list for repo '{}' not found", repo_name))?;

    let pkg_entry = pkg_list.packages.get(&version.pkgname);
    let manager_entry = get_manager_entry(pkg_entry.is_none(), selector, &version.pkgname, &pkg_list);

    let (star_path, function_name) = match (pkg_entry, manager_entry) {
        (Some(pkg), _) => (Path::new(&repo.path).join(&pkg.filename), &pkg.function_name),
        (None, Some(mgr)) => (Path::new(&repo.path).join(&mgr.filename), &mgr.function_name),
        _ => return Ok(None),
    };

    let options = extract_options(ctx.all_options, &version.pkgname);

    let dynamic_versions = if manager_entry.is_some() {
        let pkg_name = if version.pkgname.contains(':') { version.pkgname.split(':').nth(1).unwrap() } else { &version.pkgname };
        let prefix = selector.prefix.as_deref().unwrap_or_else(|| version.pkgname.split(':').next().unwrap());
        crate::starlark::runtime::execute_manager_function(
            crate::starlark::runtime::ExecutionOptions {
                path: &star_path,
                function_name,
                config: ctx.config,
                options: Some(options),
            },
            prefix,
            pkg_name,
        )?
    } else {
        crate::starlark::runtime::execute_function(
            crate::starlark::runtime::ExecutionOptions {
                path: &star_path,
                function_name,
                config: ctx.config,
                options: Some(options),
            },
            &version.pkgname,
        )?
    };

    Ok(dynamic_versions.into_iter().find(|v| v.version == version.version))
}

fn get_manager_entry<'a>(
    is_none: bool,
    selector: &PackageSelector,
    pkgname: &str,
    pkg_list: &'a crate::models::package_entry::PackageList
) -> Option<&'a crate::models::package_entry::ManagerEntry> {
    if !is_none { return None; }
    if let Some(prefix) = &selector.prefix {
        pkg_list.managers.get(prefix)
    } else if pkgname.contains(':') {
        pkg_list.managers.get(pkgname.split(':').next().unwrap())
    } else {
        pkg_list.managers.get(pkgname)
    }
}

fn extract_options(all_options: &HashMap<String, HashMap<String, serde_json::Value>>, pkgname: &str) -> HashMap<String, String> {
    let mut options = HashMap::new();
    if let Some(pkg_opts) = all_options.get(pkgname) {
        for (k, v) in pkg_opts {
            options.insert(k.clone(), match v {
                serde_json::Value::String(s) => s.clone(),
                serde_json::Value::Bool(b) => b.to_string(),
                _ => v.to_string(),
            });
        }
    }
    options
}

fn execute_pipeline(
    ctx: &BuildContext,
    pkg_ctx: &str,
    version: &VersionEntry,
    _repo_name: &str,
) -> Result<(String, HashMap<String, String>, Vec<(String, PathBuf, Vec<Export>)>)> {
    let mut current_path: Option<PathBuf> = None;
    let mut env = HashMap::new();
    let dependency_dirs = resolve_build_dependencies(ctx, version, pkg_ctx)?;

    let mut recomputed = false;
    for (i, step) in version.pipeline.iter().enumerate() {
        let mut resolved_step = step.clone();
        if let InstallStep::Run { ref mut command, .. } = resolved_step {
            *command = ctx.config.resolve_packages_dir(command);
        }

        let step_hash = hash_to_string(&resolved_step);
        let skip_cache = match step {
            InstallStep::Fetch { .. } => false, // Fetch handles its own "exists" check
            _ => ctx.config.rebuild,
        };

        if !ctx.config.force && !recomputed && !skip_cache {
            if let Some(cached) = ctx.build_cache.get_step_result(&version.pkgname, &version.version.to_string(), i, &step_hash) {
                current_path = cached.output_path;
                continue;
            }
        }

        recomputed = true;
        let step_ctx = StepContext {
            config: ctx.config,
            env: &env,
            dependency_dirs: dependency_dirs.clone(),
            pkgname: &version.pkgname,
            version: &version.version.to_string(),
            pilocal_dir: ctx.pilocal_dir,
        };

        let result_path = execute_step(&step_ctx, &resolved_step, &current_path)?;
        update_step_cache(ctx.build_cache, version, i, step_hash, &resolved_step, result_path.clone())?;
        current_path = Some(result_path);
    }

    let source_root = current_path.unwrap_or_else(|| {
        ctx.config.cache_packages_dir.join(version.pkg_dir_name())
    });

    for export in &version.exports {
        if let Export::Env { key, val } = export { env.insert(key.clone(), val.clone()); }
    }

    Ok((pkg_ctx.to_string(), env, vec![(pkg_ctx.to_string(), source_root, version.exports.clone())]))
}

fn resolve_build_dependencies(ctx: &BuildContext, version: &VersionEntry, pkg_ctx: &str) -> Result<Vec<PathBuf>> {
    let mut dirs = Vec::new();
    for dep in &version.build_dependencies {
        let selector = match PackageSelector::parse(&dep.name) {
            Some(s) => s,
            None => {
                if !dep.optional { anyhow::bail!("[{}] invalid dep selector: {}", pkg_ctx, dep.name); }
                continue;
            }
        };

        if let Some((_, dep_version, dep_repo)) = resolve::resolve_query(ctx.config, ctx.repo_config, &selector) {
            let dyn_dep = re_evaluate_version(ctx, &dep_repo, &dep_version, &selector)?;
            for export in &dyn_dep.exports {
                if let Export::Link { src, .. } = export {
                    let resolved_src = ctx.config.resolve_packages_dir(src);
                    let p = Path::new(&resolved_src);
                    if p.is_absolute() {
                        if let Some(parent) = p.parent() {
                            let parent_buf = parent.to_path_buf();
                            if !dirs.contains(&parent_buf) { dirs.push(parent_buf); }
                        }
                    }
                }
            }
        } else if !dep.optional {
            anyhow::bail!("[{}] missing required dependency: {}", pkg_ctx, dep.name);
        }
    }
    Ok(dirs)
}

fn update_step_cache(
    cache: &BuildCache,
    version: &VersionEntry,
    i: usize,
    hash: String,
    step: &InstallStep,
    result_path: PathBuf
) -> Result<()> {
    let name = match step {
        InstallStep::Fetch { name, .. } | InstallStep::Extract { name, .. } | InstallStep::Run { name, .. } => name.clone(),
    };
    cache.update_step_result(&version.pkgname, &version.version.to_string(), i, StepResult {
        name, step_hash: hash, timestamp: chrono::Utc::now().to_rfc3339(),
        output_path: Some(result_path), status: "Success".to_string(),
    })
}

fn prepare_build_sandbox(
    config: &Config,
    pkgname: &str,
    version: &str,
    homedir: &Path,
    pilocal_dir: &Path,
    env_vars: &HashMap<String, String>,
    dependency_dirs: &[PathBuf],
) -> Result<crate::services::sandbox::Bubblewrap> {
    let mut b = crate::services::sandbox::Bubblewrap::new();
    let host_home = config.get_host_home();
    let internal_pilocal = host_home.join(".pilocal");

    // System paths
    b.add_flag("--unshare-pid");
    b.add_flag("--unshare-uts");
    b.add_flag("--die-with-parent");
    b.add_bind(crate::services::sandbox::BindType::RoBind, "/usr");
    b.add_bind(crate::services::sandbox::BindType::RoBind, "/lib");
    if Path::new("/lib64").exists() {
        b.add_bind(crate::services::sandbox::BindType::RoBind, "/lib64");
    }
    b.add_bind(crate::services::sandbox::BindType::RoBind, "/bin");
    b.add_bind(crate::services::sandbox::BindType::RoBind, "/sbin");
    b.add_bind(crate::services::sandbox::BindType::RoBind, "/etc");
    b.add_bind(crate::services::sandbox::BindType::RoBind, "/sys");

    // Virtual fs
    b.add_virtual(crate::services::sandbox::BindType::Proc, "/proc");
    b.add_virtual(crate::services::sandbox::BindType::Dev, "/dev");
    b.add_virtual(crate::services::sandbox::BindType::Tmpfs, "/tmp");
    b.add_virtual(crate::services::sandbox::BindType::Tmpfs, "/run");

    // Home and caches
    std::fs::create_dir_all(homedir.join(".cache")).ok();
    std::fs::create_dir_all(homedir.join(".config")).ok();
    std::fs::create_dir_all(homedir.join(".cache").join("pi")).ok();
    std::fs::create_dir_all(homedir.join(".config").join("pi")).ok();
    b.add_map_bind(crate::services::sandbox::BindType::Bind, homedir, &host_home);

    // Mount the cave's pilocal to ~/.pilocal
    if !pilocal_dir.exists() {
        std::fs::create_dir_all(pilocal_dir).ok();
    }
    b.add_map_bind(crate::services::sandbox::BindType::Bind, pilocal_dir, &internal_pilocal);

    if config.cache_dir.exists() {
        b.add_bind(crate::services::sandbox::BindType::Bind, &config.cache_dir);
    }
    if config.config_dir.exists() {
        b.add_bind(crate::services::sandbox::BindType::RoBind, &config.config_dir);
    }

    if let Ok(runtime_dir) = env::var("XDG_RUNTIME_DIR") {
        b.add_bind(crate::services::sandbox::BindType::Bind, &runtime_dir);
        b.set_env("XDG_RUNTIME_DIR", &runtime_dir);
    }

    // Dependencies
    for dir in dependency_dirs {
        if dir.exists() {
            b.add_bind(crate::services::sandbox::BindType::RoBind, dir);
            let bin_dir = dir.join("bin");
            if bin_dir.exists() {
                b.add_env_first("PATH", bin_dir.to_str().unwrap());
            }
        }
    }

    // Environment
    b.set_env("HOME", host_home.to_str().unwrap());
    b.set_env("USER", &config.get_user());
    let pilocal_bin = internal_pilocal.join("bin");
    b.add_env_first("PATH", "/usr/bin:/bin");
    b.add_env_first("PATH", host_home.join(".local").join("bin").to_str().unwrap());
    b.add_env_first("PATH", host_home.join(".cargo").join("bin").to_str().unwrap());
    b.add_env_first("PATH", host_home.join(".mix").join("escripts").to_str().unwrap());
    b.add_env_first("PATH", pilocal_bin.to_str().unwrap());

    // Custom envs
    let resolve = |v: String| {
        v.replace("$/", &format!("{}/", internal_pilocal.display()))
         .replace("$", internal_pilocal.to_str().unwrap())
         .replace("@HOME", host_home.to_str().unwrap())
    };
    for (k, v) in env_vars {
        b.set_env(k, &resolve(v.clone()));
    }

    b.set_hostname(&format!("build-{}-{}", pkgname, version));

    Ok(b)
}

fn execute_step(ctx: &StepContext, step: &InstallStep, current_path: &Option<PathBuf>) -> Result<PathBuf> {
    match step {
        InstallStep::Fetch { url, checksum, filename, .. } => {
            let fname = filename.clone().unwrap_or_else(|| url.split('/').last().unwrap_or("download").to_string());
            let dest = ctx.config.cache_download_dir.join(fname);
            
            if dest.exists() {
                if let Some(cs) = checksum {
                    log::debug!("skipping download, file exists: {}", dest.display());
                    return Ok(dest);
                } else {
                    log::debug!("skipping download, file exists: {}", dest.display());
                    return Ok(dest);
                }
            }
            Downloader::download_to_file(url, &dest, checksum.as_deref())?;
            Ok(dest)
        }
        InstallStep::Extract { .. } => {
            let src = current_path.as_ref().context("Extract requires a Fetch step")?;
            let pkg_dir = format!("{}-extracted", sanitize_name(&format!("{}-{}", ctx.pkgname, ctx.version)));
            let dest = ctx.config.cache_packages_dir.join(pkg_dir);

            if dest.exists() && !ctx.config.rebuild && !ctx.config.force {
                log::debug!("skipping extraction, directory exists: {}", dest.display());
                return Ok(dest);
            }

            if dest.exists() {
                let _ = fs::remove_dir_all(&dest);
            }
            Unarchiver::unarchive(src, &dest)?;
            Ok(dest)
        }
        InstallStep::Run { command, cwd, .. } => {
            let default_base = ctx.config.cache_packages_dir.join(sanitize_name(&format!("{}-{}", ctx.pkgname, ctx.version)));
            let base_dir = cwd.as_ref().map(|c| current_path.as_ref().unwrap_or(&default_base).join(c)).unwrap_or_else(|| current_path.clone().unwrap_or(default_base));
            fs::create_dir_all(&base_dir).ok();

            // Create a temporary home directory for manager execution
            let tmp_home = tempfile::tempdir().context("Failed to create temporary home directory")?;

            let mut b = prepare_build_sandbox(
                ctx.config,
                ctx.pkgname,
                ctx.version,
                tmp_home.path(),
                ctx.pilocal_dir,
                ctx.env,
                &ctx.dependency_dirs,
            )?;
            
            b.set_cwd(&base_dir);
            b.set_command("/bin/bash", &[String::from("-c"), command.clone()]);
            b.spawn().with_context(|| format!("Failed to execute command: {}", command))?;

            Ok(base_dir)
        }
    }
}
