use anyhow::{Context, Result};
use std::fs::{self, File};
use std::path::Path;
use flate2::read::GzDecoder;
use xz2::read::XzDecoder;
use tar::Archive;
use zip::ZipArchive;

pub struct Unarchiver;

impl Unarchiver {
    pub fn unarchive(src: &Path, dest: &Path) -> Result<()> {
        let marker_file = dest.join(".unarchived");
        if marker_file.exists() {
            return Ok(());
        }

        fs::create_dir_all(dest).context("Failed to create destination directory")?;

        let filename = src.file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("");

        if filename.ends_with(".tar.gz") || filename.ends_with(".tgz") {
            let file = File::open(src)?;
            let tar = GzDecoder::new(file);
            let mut archive = Archive::new(tar);
            archive.unpack(dest).context("Failed to unpack tar.gz")?;
        } else if filename.ends_with(".tar.xz") {
            let file = File::open(src)?;
            let tar = XzDecoder::new(file);
            let mut archive = Archive::new(tar);
            archive.unpack(dest).context("Failed to unpack tar.xz")?;
        } else if filename.ends_with(".zip") {
            let file = File::open(src)?;
            let mut archive = ZipArchive::new(file).context("Failed to open zip archive")?;
            archive.extract(dest).context("Failed to extract zip archive")?;
        } else {
            return Err(anyhow::anyhow!("Unsupported archive format: {}", filename));
        }

        fs::write(&marker_file, "").context("Failed to create unarchive marker file")?;
        log::debug!("[{}] unarchived to {}", filename, dest.display());
        Ok(())
    }
}
