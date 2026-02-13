use anyhow::{Context, Result};
use std::fs::File;
use std::io::{Read, Write};
use std::path::Path;
use std::time::{Duration, Instant};
use ureq::Agent;
use ureq::config::IpFamily;
use sha2::{Sha256, Sha512, Digest};
use sha1::Sha1;
use hex;

pub struct Downloader;

impl Downloader {
    pub fn download(url: &str) -> Result<String> {
        let config = Agent::config_builder()
            .ip_family(IpFamily::Ipv4Only)
            .build();
        let agent = Agent::new_with_config(config);

        let response = agent.get(url).call()?;
        let mut reader = response.into_body().into_reader();
        let mut content = Vec::new();
        reader.read_to_end(&mut content)?;

        Ok(String::from_utf8(content)?)
    }

    pub fn download_to_file(url: &str, dest: &Path, expected_checksum: Option<&str>) -> Result<()> {
        if let Some(parent) = dest.parent() {
            std::fs::create_dir_all(parent).context("Failed to create download directory")?;
        }

        // If file already exists and checksum matches, skip
        if dest.exists() && expected_checksum.is_some() {
            let expected = expected_checksum.unwrap();
            if let Ok(actual_checksum) = Self::calculate_checksum(dest, expected.len()) {
                if actual_checksum == expected {
                    log::info!("[{}] skip, matches checksum", dest.display());
                    return Ok(());
                }
            }
        }

        let config = Agent::config_builder()
            .ip_family(IpFamily::Ipv4Only)
            .build();
        let agent = Agent::new_with_config(config);

        log::info!("[{}] fetching", url);
        let response = agent.get(url).call()?;

        let content_length = response
            .headers()
            .get("content-length")
            .and_then(|h| h.to_str().ok())
            .and_then(|s| s.parse::<u64>().ok());

        let filename = url.split('/').last().unwrap_or("unknown");

        let mut reader = response.into_body().into_reader();
        let mut file = File::create(dest).context("Failed to create destination file")?;
        
        let mut buffer = [0; 8192];
        let mut downloaded_bytes: u64 = 0;
        let mut last_report_time = Instant::now();
        let start_time = Instant::now();

        loop {
            let bytes_read = reader.read(&mut buffer)?;
            if bytes_read == 0 {
                break;
            }

            file.write_all(&buffer[..bytes_read])?;
            downloaded_bytes += bytes_read as u64;

            let now = Instant::now();
            if now.duration_since(last_report_time) >= Duration::from_secs(5) {
                let total_elapsed = now.duration_since(start_time).as_secs_f64();
                let bandwidth = if total_elapsed > 0.0 {
                    (downloaded_bytes as f64) / total_elapsed
                } else {
                    0.0
                };

                let expected_str = match content_length {
                    Some(len) => len.to_string(),
                    None => "???".to_string(),
                };

                log::debug!(
                    "[{}] recv {}/{} ({:.2} KB/s)",
                    filename, downloaded_bytes, expected_str, bandwidth / 1024.0
                );
                last_report_time = now;
            }
        }

        if let Some(expected) = expected_checksum {
            let actual = Self::calculate_checksum(dest, expected.len())?;
            if actual != expected {
                return Err(anyhow::anyhow!(
                    "[{}] checksum mismatch: got {}, want {}",
                    url, actual, expected
                ));
            }
            log::debug!("[{}] checksum ok", filename);
        }

        Ok(())
    }

    fn calculate_checksum(path: &Path, expected_len: usize) -> Result<String> {
        let mut file = File::open(path)?;
        let mut buffer = [0; 8192];

        match expected_len {
            40 => {
                let mut hasher = Sha1::new();
                loop {
                    let n = file.read(&mut buffer)?;
                    if n == 0 {
                        break;
                    }
                    hasher.update(&buffer[..n]);
                }
                Ok(hex::encode(hasher.finalize()))
            }
            64 => {
                let mut hasher = Sha256::new();
                loop {
                    let n = file.read(&mut buffer)?;
                    if n == 0 {
                        break;
                    }
                    hasher.update(&buffer[..n]);
                }
                Ok(hex::encode(hasher.finalize()))
            }
            128 => {
                let mut hasher = Sha512::new();
                loop {
                    let n = file.read(&mut buffer)?;
                    if n == 0 {
                        break;
                    }
                    hasher.update(&buffer[..n]);
                }
                Ok(hex::encode(hasher.finalize()))
            }
            _ => Err(anyhow::anyhow!(
                "Unsupported checksum length: {}. Expected 40 (SHA-1), 64 (SHA-256), or 128 (SHA-512).",
                expected_len
            )),
        }
    }
}
