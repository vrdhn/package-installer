use anyhow::{Context, Result};
use std::fs::File;
use std::io::{Read, Write};
use std::path::Path;
use std::time::{Duration, Instant};
use ureq::Agent;
use ureq::config::IpFamily;
use crate::utils::crypto::calculate_file_checksum;

pub struct Downloader;

impl Downloader {
    pub fn download(url: &str) -> Result<String> {
        let agent = Self::create_agent();
        let response = agent.get(url).call()?;
        let mut reader = response.into_body().into_reader();
        let mut content = Vec::new();
        reader.read_to_end(&mut content)?;

        Ok(String::from_utf8(content)?)
    }

    pub fn download_to_file(url: &str, dest: &Path, expected_checksum: Option<&str>) -> Result<()> {
        Self::prepare_directory(dest)?;

        if Self::is_file_ready(dest, expected_checksum) {
            return Ok(());
        }

        let agent = Self::create_agent();
        log::info!("[{}] fetching", url);
        let response = agent.get(url).call()?;

        let content_length = Self::get_content_length(&response);
        let filename = url.split('/').last().unwrap_or("unknown");

        // Download to a temporary file in the same directory to ensure atomic rename
        let parent = dest.parent().context("Destination has no parent directory")?;
        let mut tmp_file = tempfile::NamedTempFile::new_in(parent)
            .context("Failed to create temporary download file")?;

        Self::stream_to_file(response.into_body().into_reader(), tmp_file.as_file_mut(), content_length, filename)?;

        Self::verify_checksum(url, tmp_file.path(), expected_checksum, filename)?;

        // Persist the temporary file to the final destination
        tmp_file.persist(dest).map_err(|e| {
            anyhow::anyhow!("Failed to persist download to {}: {}", dest.display(), e.error)
        })?;

        Ok(())
    }

    fn create_agent() -> Agent {
        let config = Agent::config_builder()
            .ip_family(IpFamily::Ipv4Only)
            .build();
        Agent::new_with_config(config)
    }

    fn prepare_directory(dest: &Path) -> Result<()> {
        if let Some(parent) = dest.parent() {
            std::fs::create_dir_all(parent).context("Failed to create download directory")?;
        }
        Ok(())
    }

    fn is_file_ready(dest: &Path, expected_checksum: Option<&str>) -> bool {
        if let (true, Some(expected)) = (dest.exists(), expected_checksum) {
            if let Ok(actual) = calculate_file_checksum(dest, expected.len()) {
                if actual == expected {
                    log::info!("[{}] skip, matches checksum", dest.display());
                    return true;
                }
            }
        }
        false
    }

    fn get_content_length<T>(response: &ureq::http::Response<T>) -> Option<u64> {
        let headers = response.headers();
        headers.get("content-length")
            .and_then(|h| h.to_str().ok())
            .and_then(|s: &str| s.parse::<u64>().ok())
    }

    fn stream_to_file(mut reader: impl Read, file: &mut File, total_size: Option<u64>, filename: &str) -> Result<()> {
        let mut buffer = [0; 8192];
        let mut downloaded: u64 = 0;
        let mut last_report = Instant::now();
        let start_time = Instant::now();

        loop {
            let n = reader.read(&mut buffer)?;
            if n == 0 { break; }

            file.write_all(&buffer[..n])?;
            downloaded += n as u64;

            if last_report.elapsed() >= Duration::from_secs(5) {
                Self::report_progress(filename, downloaded, total_size, start_time.elapsed());
                last_report = Instant::now();
            }
        }
        Ok(())
    }

    fn report_progress(filename: &str, downloaded: u64, total: Option<u64>, elapsed: Duration) {
        let bandwidth = downloaded as f64 / elapsed.as_secs_f64();
        let total_str = total.map(|t| t.to_string()).unwrap_or_else(|| "???".to_string());
        log::debug!(
            "[{}] recv {}/{} ({:.2} KB/s)",
            filename, downloaded, total_str, bandwidth / 1024.0
        );
    }

    fn verify_checksum(url: &str, dest: &Path, expected: Option<&str>, filename: &str) -> Result<()> {
        if let Some(expected) = expected {
            let actual = calculate_file_checksum(dest, expected.len())?;
            if actual != expected {
                return Err(anyhow::anyhow!("[{}] checksum mismatch: got {}, want {}", url, actual, expected));
            }
            log::debug!("[{}] checksum ok", filename);
        }
        Ok(())
    }
}
