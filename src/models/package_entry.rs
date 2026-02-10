use allocative::Allocative;
use serde::Serialize;

#[derive(Debug, Clone, Allocative, Serialize)]
pub struct PackageEntry {
    pub regexp: String,
    pub function_name: String,
    pub filename: String,
}
