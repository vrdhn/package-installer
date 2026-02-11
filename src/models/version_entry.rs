use allocative::Allocative;
use serde::Serialize;

#[derive(Debug, Clone, Serialize, Allocative)]
pub struct VersionEntry {
    pub pkgname: String,
    pub version: String,
    pub release_date: String,
    pub release_type: String,
    pub url: String,
    pub filename: String,
    pub checksum: String,
    pub checksum_url: String,
}
