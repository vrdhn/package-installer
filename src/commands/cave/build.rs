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
use anyhow::Context;
use rayon::prelude::*;
use log::error;
use walkdir::WalkDir;

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

    println!("Building cave {} (variant {:?})...", cave.name, variant);

    let repo_config = Repositories::get_all(config);

    let results: Vec<anyhow::Result<(String, PathBuf, std::collections::HashMap<String, String>)>> = settings.packages
        .par_iter()
        .map(|query| {
            let selector = PackageSelector::parse(query)
                .ok_or_else(|| anyhow::anyhow!("Invalid selector: {}", query))?;

            let (full_name, version, repo_uuid) = resolve::resolve_query(config, repo_config, &selector)
                .ok_or_else(|| anyhow::anyhow!("Package not found: {}", query))?;

            let download_dest = config.download_dir.join(&version.filename);
            let checksum = if version.checksum.is_empty() { None } else { Some(version.checksum.as_str()) };

            Downloader::download_to_file(&version.url, &download_dest, checksum)?;

            let pkg_dir_name = format!("{}-{}-{}", sanitize_name(&version.pkgname), sanitize_name(&version.version), repo_uuid);
            let extract_dest = config.packages_dir.join(pkg_dir_name);

            Unarchiver::unarchive(&download_dest, &extract_dest)?;

            Ok((full_name, extract_dest, version.filemap))
        })
        .collect();

    let mut all_filemap = Vec::new();
    for res in results {
        match res {
            Ok((full_name, extract_dest, filemap)) => {
                println!("Resolved {} to {}", full_name, extract_dest.display());
                all_filemap.push((extract_dest, filemap));
            }
            Err(e) => {
                error!("Build failed for a package: {}", e);
            }
        }
    }

    let home_dir = &cave.homedir;
    let pitree_dir = home_dir.join(".pitree");
    fs::create_dir_all(&pitree_dir).expect("Failed to create .pitree directory");

    println!("Applying filemap to pitree: {}", pitree_dir.display());

    for (pkg_dir, filemap) in all_filemap {
        for (src_pattern, dest_rel) in filemap {
            if let Err(e) = apply_filemap_entry(&pkg_dir, &pitree_dir, &src_pattern, &dest_rel) {
                error!("Failed to apply filemap entry '{}' for {}: {}", src_pattern, pkg_dir.display(), e);
            }
        }
    }
    
    println!("Build finished successfully.");
}

fn apply_filemap_entry(pkg_dir: &Path, pitree_dir: &Path, src_pattern: &str, dest_rel: &str) -> anyhow::Result<()> {
    if src_pattern.contains('*') {
        // Glob-like resolution using walkdir (simple * at end support)
        let base_pattern = src_pattern.strip_suffix("*").unwrap_or(src_pattern);
        let search_path = pkg_dir.join(base_pattern);
        
        if !search_path.exists() {
            return Err(anyhow::anyhow!("Source pattern base path does not exist: {}", search_path.display()));
        }

        let mut matched = false;
        if search_path.is_dir() {
            for entry in WalkDir::new(&search_path).max_depth(1).into_iter().filter_map(|e| e.ok()) {
                if entry.path() == search_path { continue; }
                let _rel_to_pkg = entry.path().strip_prefix(pkg_dir).unwrap();
                let file_name = entry.file_name();
                
                let target_dest = pitree_dir.join(dest_rel).join(file_name);
                println!("  Symlinking {} -> {}", target_dest.display(), entry.path().display());
                create_symlink(entry.path(), &target_dest)?;
                matched = true;
            }
        }

        if !matched {
            return Err(anyhow::anyhow!("Source pattern '{}' did not match any files in {}", src_pattern, pkg_dir.display()));
        }
    } else {
        let src_path = pkg_dir.join(src_pattern);
        let dest_path = pitree_dir.join(dest_rel);
        
        if !src_path.exists() {
            return Err(anyhow::anyhow!("Source path does not exist: {}", src_path.display()));
        }

        let final_dest = if dest_rel.ends_with('/') || dest_path.is_dir() {
            let file_name = src_path.file_name().ok_or_else(|| anyhow::anyhow!("Invalid source filename"))?;
            dest_path.join(file_name)
        } else {
            dest_path
        };

        println!("  Symlinking {} -> {}", final_dest.display(), src_path.display());
        create_symlink(&src_path, &final_dest)?;
    }
    Ok(())
}

fn create_symlink(src: &Path, dest: &Path) -> anyhow::Result<()> {
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
