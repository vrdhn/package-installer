use allocative::Allocative;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Allocative, Serialize, Deserialize)]
pub struct PackageEntry {
    pub name: String,
    pub function_name: String,
    pub filename: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PackageList {
    pub packages: Vec<PackageEntry>,
}
