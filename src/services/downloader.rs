use anyhow::Result;

pub struct Downloader;

impl Downloader {
    pub fn download(url: &str) -> Result<String> {
        let response = ureq::get(url).call()?;
        let body = response.into_body().read_to_string()?;
        Ok(body)
    }
}
