use crate::models::config::Config;
use crate::models::cave::Cave;
use crate::models::selector::PackageSelector;
use crate::models::repository::Repositories;
use crate::commands::package::resolve;
use crate::services::downloader::Downloader;
use crate::services::unarchiver::Unarchiver;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use anyhow::{Context, Result};
use rayon::prelude::*;
use log::error;
use walkdir::WalkDir;

use crate::models::version_entry::{ManagerCommand};

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

    let results: Vec<Result<(String, String, Option<PathBuf>, std::collections::HashMap<String, String>, std::collections::HashMap<String, String>, ManagerCommand)>> = settings.packages
        .par_iter()
        .map(|query| {
            let selector = PackageSelector::parse(query)
                .ok_or_else(|| anyhow::anyhow!("Invalid selector: {}", query))?;

            let (full_name, version, repo_name) = resolve::resolve_query(config, repo_config, &selector)
                .ok_or_else(|| anyhow::anyhow!("Package not found: {}", query))?;

            let pkg_ctx = format!("{}:{}={}", repo_name, full_name, version.version);

            if let ManagerCommand::Custom(_) = version.manager_command {
                // Skip download/unarchive for managed packages
                return Ok((pkg_ctx, full_name, None, version.filemap, version.env, version.manager_command));
            }

            let download_dest = config.download_dir.join(&version.filename);
            let checksum = if version.checksum.is_empty() { None } else { Some(version.checksum.as_str()) };

            Downloader::download_to_file(&version.url, &download_dest, checksum)?;

            let pkg_dir_name = format!("{}-{}-{}", sanitize_name(&version.pkgname), sanitize_name(&version.version), repo_name);
            let extract_dest = config.packages_dir.join(pkg_dir_name);

            Unarchiver::unarchive(&download_dest, &extract_dest)?;

            Ok((pkg_ctx, full_name, Some(extract_dest), version.filemap, version.env, version.manager_command))
        })
        .collect();

    let mut all_filemap = Vec::new();
    let mut all_env = std::collections::HashMap::new();
    let mut manager_commands = Vec::new();

    for res in results {
        match res {
            Ok((pkg_ctx, _full_name, extract_dest, filemap, env, manager_cmd)) => {
                if let Some(ref dest) = extract_dest {
                    log::debug!("[{}] resolved to {}", pkg_ctx, dest.display());
                    all_filemap.push((pkg_ctx, dest.clone(), filemap));
                } else {
                    log::debug!("[{}] managed pkg, skipping extraction", pkg_ctx);
                }

                for (k, v) in env {
                    all_env.insert(k, v);
                }
                match manager_cmd {
                    ManagerCommand::Custom(cmd) => manager_commands.push(cmd),
                    ManagerCommand::Auto => {}
                }
            }
            Err(e) => {
                error!("build failed: {}", e);
                return Err(e);
            }
        }
    }

    let pilocal_dir = config.pilocal_path(&cave.name, variant);
    fs::create_dir_all(&pilocal_dir).context("Failed to create .pilocal directory")?;

    log::debug!("[{}] applying filemap: {}", cave.name, pilocal_dir.display());

    for (pkg_ctx, pkg_dir, filemap) in all_filemap {
        for (src_pattern, dest_rel) in filemap {
            apply_filemap_entry(&pkg_ctx, &pkg_dir, &pilocal_dir, &src_pattern, &dest_rel)
                .with_context(|| format!("Failed to apply filemap entry '{}' for {}", src_pattern, pkg_dir.display()))?;
        }
    }

    if !manager_commands.is_empty() {
        run_manager_commands(config, cave, variant, &all_env, manager_commands)?;
    }
    
    log::info!("[{}] build success", cave.name);
    Ok(all_env)
}

fn run_manager_commands(
    config: &Config,
    cave: &Cave,
    variant: Option<&str>,
    all_env: &std::collections::HashMap<String, String>,
    commands: Vec<String>,
) -> Result<()> {
    log::info!("[{}] running manager commands", cave.name);

    // Generate install script
    let mut script_content = String::from("#!/bin/bash\nset -e\n");
    for cmd in commands {
        script_content.push_str(&cmd);
        script_content.push('\n');
    }

    let script_path = cave.workspace.join(".pi_install.sh");
    fs::write(&script_path, script_content).context("Failed to write install script")?;

    let mut b = crate::commands::cave::run::prepare_sandbox(config, cave, variant, all_env.clone(), true)?;
    b.set_command("/bin/bash", &[String::from(".pi_install.sh")]);
    
    let result = b.spawn();
    
    // Cleanup script
    let _ = fs::remove_file(&script_path);
    
    result.context("Failed to execute manager commands in sandbox")
}

fn apply_filemap_entry(pkg_ctx: &str, pkg_dir: &Path, pilocal_dir: &Path, src_pattern: &str, dest_rel: &str) -> Result<()> {
    if src_pattern.contains('*') {
        // Glob-like resolution using walkdir (simple * at end support)
        let base_pattern = src_pattern.strip_suffix("*").unwrap_or(src_pattern);
        let search_path = pkg_dir.join(base_pattern);
        
        if !search_path.exists() {
            return Err(anyhow::anyhow!("[{}] source pattern missing: {}", pkg_ctx, search_path.display()));
        }

        let mut matched = false;
        if search_path.is_dir() {
            for entry in WalkDir::new(&search_path).max_depth(1).into_iter().filter_map(|e| e.ok()) {
                if entry.path() == search_path { continue; }
                let _rel_to_pkg = entry.path().strip_prefix(pkg_dir).unwrap();
                let file_name = entry.file_name();
                
                let target_dest = pilocal_dir.join(dest_rel).join(file_name);
                log::trace!(
                    "[{}] link {} -> {}",
                    pkg_ctx,
                    target_dest.strip_prefix(pilocal_dir).unwrap_or(&target_dest).display(),
                    entry.file_name().to_string_lossy()
                );
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

        log::trace!(
            "[{}] link {} -> {}",
            pkg_ctx,
            final_dest.strip_prefix(pilocal_dir).unwrap_or(&final_dest).display(),
            src_path.file_name().map(|n| n.to_string_lossy()).unwrap_or_else(|| src_path.to_string_lossy())
        );
        create_symlink(&src_path, &final_dest)?;
    }
    Ok(())
}

fn create_symlink(src: &Path, dest: &Path) -> Result<()> {
    if let Some(parent) = dest.parent() {
        fs::create_dir_all(parent).context("Failed to create parent directory for symlink")?;
    }
    
    if dest.exists() || dest.is_symlink() {
        // Remove existing symlink or file
        let metadata = fs::symlink_metadata(dest).context("Failed to get metadata for existing destination")?;
        if metadata.is_dir() && !metadata.is_symlink() {
            fs::remove_dir_all(dest).context("Failed to remove existing directory at symlink destination")?;
        } else {
            fs::remove_file(dest).context("Failed to remove existing file/symlink at destination")?;
        }
    }

    #[cfg(unix)]
    {
        use std::os::unix::fs::symlink;
        symlink(src, dest).with_context(|| format!("Failed to create unix symlink {} -> {}", dest.display(), src.display()))?;
    }
    #[cfg(windows)]
    {
        use std::os::windows::fs::{symlink_file, symlink_dir};
        let res = if src.is_dir() {
            symlink_dir(src, dest)
        } else {
            symlink_file(src, dest)
        };
        res.with_context(|| format!("Failed to create windows symlink {} -> {}", dest.display(), src.display()))?;
    }
    Ok(())
}

fn sanitize_name(name: &str) -> String {
    name.replace(['/', '\\', ' ', ':'], "_")
}
