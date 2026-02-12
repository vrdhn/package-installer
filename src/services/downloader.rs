use anyhow::Result;
use std::io::Read;
use std::time::{Duration, Instant};
use ureq::Agent;
use ureq::config::IpFamily;

pub struct Downloader;

impl Downloader {
    pub fn download(url: &str) -> Result<String> {
        let config = Agent::config_builder()
            .ip_family(IpFamily::Ipv4Only)
            .build();
        let agent = Agent::new_with_config(config);

        let response = agent.get(url).call()?;

        let content_length = response
            .headers()
            .get("content-length")
            .and_then(|h| h.to_str().ok())
            .and_then(|s| s.parse::<u64>().ok());

        let filename = url.split('/').last().unwrap_or("unknown");

        let mut reader = response.into_body().into_reader();
        let mut buffer = [0; 8192];
        let mut downloaded_bytes: u64 = 0;
        let mut last_report_time = Instant::now();
        let start_time = Instant::now();
        let mut content = Vec::new();

        loop {
            let bytes_read = reader.read(&mut buffer)?;
            if bytes_read == 0 {
                break;
            }

            downloaded_bytes += bytes_read as u64;
            content.extend_from_slice(&buffer[..bytes_read]);

            let now = Instant::now();
            if now.duration_since(last_report_time) >= Duration::from_secs(10) {
                let total_elapsed = now.duration_since(start_time).as_secs_f64();
                let bandwidth = if total_elapsed > 0.0 {
                    (downloaded_bytes as f64) / total_elapsed
                } else {
                    0.0
                };

                let expected_str = match content_length {
                    Some(len) => len.to_string(),
                    None => "unknown".to_string(),
                };

                println!(
                    "Downloading {}: received {} bytes, expected {} bytes, bandwidth {:.2} KB/s",
                    filename, downloaded_bytes, expected_str, bandwidth / 1024.0
                );
                last_report_time = now;
            }
        }

        Ok(String::from_utf8(content)?)
    }
}
