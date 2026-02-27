use crate::models::config::State;
use crate::models::package_entry::{ManagerEntry, PackageEntry};
use crate::models::version_entry::VersionEntry;
use crate::models::types::{OS, Arch};
use allocative::{Allocative, Key, Visitor};
use parking_lot::RwLock;
use serde::Serialize;
use starlark::any::ProvidesStaticType;
use starlark::values::{AllocValue, Heap, StarlarkValue, Value, starlark_value};
use std::fmt::{self, Display};
use std::path::PathBuf;
use std::sync::Arc;
use std::collections::HashMap;

/// The Context struct serves as the bridge between the Rust host and the Starlark guest environment.
///
/// It is used for:
/// 1. Starlark Script Isolation: Each Starlark file (recipe) needs to know its own context,
///    such as its filename and the options passed to it, without polluting a global state.
/// 2. API Access: Starlark API functions (like add_package, download, or get_os) use the
///    Context to know where to save results and where the filesystem boundaries are.
/// 3. State Accumulation: During evaluation, Starlark scripts populate the packages and
///    managers fields. Rust then extracts these to know what the script defined.
/// 4. Type Safety & Serialization: It can be safely embedded into the Starlark Heap and
///    serialized for debugging or caching.
#[derive(Debug, ProvidesStaticType, Serialize)]
pub struct Context {
    pub os: OS,
    pub arch: Arch,
    pub filename: String,
    pub meta_dir: PathBuf,
    pub download_dir: PathBuf,
    pub packages_dir: PathBuf,
    pub force: bool,
    pub packages: RwLock<Vec<PackageEntry>>,
    pub managers: RwLock<Vec<ManagerEntry>>,
    pub versions: RwLock<Vec<VersionEntry>>,
    pub options: HashMap<String, String>,
    #[serde(skip)]
    pub state: Arc<State>,
}

impl Context {
    pub fn new(filename: String, meta_dir: PathBuf, download_dir: PathBuf, packages_dir: PathBuf, force: bool, state: Arc<State>) -> Self {
        Self {
            os: OS::default(),
            arch: Arch::default(),
            filename,
            meta_dir,
            download_dir,
            packages_dir,
            force,
            packages: RwLock::new(Vec::new()),
            managers: RwLock::new(Vec::new()),
            versions: RwLock::new(Vec::new()),
            options: HashMap::new(),
            state,
        }
    }

    pub fn with_options(mut self, options: HashMap<String, String>) -> Self {
        self.options = options;
        self
    }

    pub fn display_name(&self) -> String {
        let p = self.filename.split(':').next().unwrap_or(&self.filename);
        PathBuf::from(p)
            .file_stem()
            .map(|s| s.to_string_lossy().to_string())
            .unwrap_or_else(|| p.to_string())
    }
}

impl Allocative for Context {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let mut visitor = visitor.enter_self_sized::<Self>();
        visitor.visit_field::<String>(Key::new("os"), &self.os.to_string());
        visitor.visit_field::<String>(Key::new("arch"), &self.arch.to_string());
        visitor.visit_field::<String>(Key::new("filename"), &self.filename);
        visitor.visit_field::<String>(
            Key::new("meta_dir"),
            &self.meta_dir.to_string_lossy().to_string(),
        );
        visitor.visit_field::<String>(
            Key::new("download_dir"),
            &self.download_dir.to_string_lossy().to_string(),
        );
        visitor.visit_field::<String>(
            Key::new("packages_dir"),
            &self.packages_dir.to_string_lossy().to_string(),
        );
        visitor.visit_field::<bool>(Key::new("force"), &self.force);
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
