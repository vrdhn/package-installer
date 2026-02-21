use anyhow::{Context, Result};
use std::fs;
use std::path::Path;
use walkdir::WalkDir;

pub fn apply_filemap_entry(pkg_ctx: &str, pkg_dir: &Path, pilocal_dir: &Path, src_pattern: &str, dest_rel: &str) -> Result<()> {
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
        fs::create_dir_all(parent).context("Failed to create parent directory for symlink")?;
    }
    if dest.exists() || dest.is_symlink() {
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
