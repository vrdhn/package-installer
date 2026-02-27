use anyhow::{Context, Result};
use std::fs;
use std::path::{Path, PathBuf};
use walkdir::WalkDir;

/// Options for mapping files from a package directory to a cave's .pilocal directory.
pub struct FileMapOptions<'a> {
    pub pkg_ctx: &'a str,
    pub pkg_dir: &'a Path,
    pub pilocal_dir: &'a Path,
    pub src_pattern: &'a str,
    pub dest_rel: &'a str,
}

/// Applies a file mapping entry, creating symlinks for matched files.
/// Example pkg_dir: "/home/user/.cache/pi/packages/rust-1.70.0"
/// Example pilocal_dir: "/home/user/.cache/pi/pilocals/my-cave"
pub fn apply_filemap_entry(opts: FileMapOptions) -> Result<()> {
    let is_glob = opts.src_pattern.contains('*');
    let base_pattern = if is_glob {
        opts.src_pattern.strip_suffix("*").unwrap_or(opts.src_pattern)
    } else {
        opts.src_pattern
    };

    let search_path = resolve_src_path(opts.pkg_dir, base_pattern);
    if !search_path.exists() {
        log::debug!("[{}] optional source missing: {}", opts.pkg_ctx, search_path.display());
        return Ok(());
    }

    if is_glob {
        apply_glob_filemap(&opts, &search_path)
    } else {
        apply_single_filemap(&opts, &search_path)
    }
}

fn apply_glob_filemap(opts: &FileMapOptions, search_path: &Path) -> Result<()> {
    let mut matched = false;
    if search_path.is_dir() {
        for entry in WalkDir::new(search_path).max_depth(1).into_iter().filter_map(|e| e.ok()) {
            if entry.path() == search_path { continue; }
            let target_dest = opts.pilocal_dir.join(opts.dest_rel).join(entry.file_name());
            create_symlink(entry.path(), &target_dest)?;
            matched = true;
        }
    }
    if !matched {
        log::debug!("[{}] pattern '{}' no match in {}", opts.pkg_ctx, opts.src_pattern, search_path.display());
    }
    Ok(())
}

fn apply_single_filemap(opts: &FileMapOptions, search_path: &Path) -> Result<()> {
    let dest_path = opts.pilocal_dir.join(opts.dest_rel);
    let final_dest = if opts.dest_rel.ends_with('/') || dest_path.is_dir() {
        let file_name = search_path.file_name().ok_or_else(|| anyhow::anyhow!("Invalid source filename"))?;
        dest_path.join(file_name)
    } else {
        dest_path
    };
    create_symlink(search_path, &final_dest)
}

fn resolve_src_path(pkg_dir: &Path, pattern: &str) -> PathBuf {
    let p = Path::new(pattern);
    if p.is_absolute() { p.to_path_buf() } else { pkg_dir.join(pattern) }
}

fn create_symlink(src: &Path, dest: &Path) -> Result<()> {
    if let Some(parent) = dest.parent() {
        fs::create_dir_all(parent).context("Failed to create parent directory for symlink")?;
    }
    ensure_destination_clear(dest)?;
    
    #[cfg(unix)]
    {
        use std::os::unix::fs::symlink;
        symlink(src, dest).with_context(|| format!("Failed to create unix symlink {} -> {}", dest.display(), src.display()))?;
    }
    #[cfg(windows)]
    {
        use std::os::windows::fs::{symlink_file, symlink_dir};
        let res = if src.is_dir() { symlink_dir(src, dest) } else { symlink_file(src, dest) };
        res.with_context(|| format!("Failed to create windows symlink {} -> {}", dest.display(), src.display()))?;
    }
    Ok(())
}

fn ensure_destination_clear(dest: &Path) -> Result<()> {
    if dest.exists() || dest.is_symlink() {
        let metadata = fs::symlink_metadata(dest).context("Failed to get metadata for existing destination")?;
        if metadata.is_dir() && !metadata.is_symlink() {
            fs::remove_dir_all(dest).context("Failed to remove existing directory")?;
        } else {
            fs::remove_file(dest).context("Failed to remove existing file/symlink")?;
        }
    }
    Ok(())
}
