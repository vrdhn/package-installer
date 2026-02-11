use crate::models::package_entry::PackageEntry;
use crate::models::version_entry::VersionEntry;
use allocative::{Allocative, Key, Visitor};
use parking_lot::RwLock;
use serde::Serialize;
use starlark::any::ProvidesStaticType;
use starlark::values::{AllocValue, Heap, StarlarkValue, Value, starlark_value};
use std::env;
use std::fmt::{self, Display};
use std::path::PathBuf;

#[derive(Debug, ProvidesStaticType, Serialize)]
pub struct Context {
    pub os: String,
    pub arch: String,
    pub filename: String,
    pub download_dir: PathBuf,
    pub packages: RwLock<Vec<PackageEntry>>,
    pub versions: RwLock<Vec<VersionEntry>>,
}

impl Context {
    pub fn new(filename: String, download_dir: PathBuf) -> Self {
        Self {
            os: env::consts::OS.to_string(),
            arch: env::consts::ARCH.to_string(),
            filename,
            download_dir,
            packages: RwLock::new(Vec::new()),
            versions: RwLock::new(Vec::new()),
        }
    }
}

impl Allocative for Context {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let mut visitor = visitor.enter_self_sized::<Self>();
        visitor.visit_field::<String>(Key::new("os"), &self.os);
        visitor.visit_field::<String>(Key::new("arch"), &self.arch);
        visitor.visit_field::<String>(Key::new("filename"), &self.filename);
        visitor.visit_field::<String>(
            Key::new("download_dir"),
            &self.download_dir.to_string_lossy().to_string(),
        );
        visitor.exit();
    }
}

impl Display for Context {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "Context(os={}, arch={}, filename={}, packages_count={}, versions_count={})",
            self.os,
            self.arch,
            self.filename,
            self.packages.read().len(),
            self.versions.read().len()
        )
    }
}

#[starlark_value(type = "Context")]
impl<'v> StarlarkValue<'v> for Context {}

impl<'v> AllocValue<'v> for Context {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}
