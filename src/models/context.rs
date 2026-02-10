use starlark::values::{starlark_value, StarlarkValue, Value, AllocValue, Heap};
use starlark::any::ProvidesStaticType;
use std::env;
use std::fmt::{self, Display};
use parking_lot::RwLock;
use allocative::{Allocative, Visitor, Key};
use serde::Serialize;
use crate::models::package_entry::PackageEntry;

#[derive(Debug, ProvidesStaticType, Serialize)]
pub struct Context {
    pub os: String,
    pub arch: String,
    pub filename: String,
    pub packages: RwLock<Vec<PackageEntry>>,
}

impl Context {
    pub fn new(filename: String) -> Self {
        Self {
            os: env::consts::OS.to_string(),
            arch: env::consts::ARCH.to_string(),
            filename,
            packages: RwLock::new(Vec::new()),
        }
    }
}

impl Allocative for Context {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let mut visitor = visitor.enter_self_sized::<Self>();
        visitor.visit_field::<String>(Key::new("os"), &self.os);
        visitor.visit_field::<String>(Key::new("arch"), &self.arch);
        visitor.visit_field::<String>(Key::new("filename"), &self.filename);
        visitor.exit();
    }
}

impl Display for Context {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Context(os={}, arch={}, filename={}, packages_count={})", 
            self.os, self.arch, self.filename, self.packages.read().len())
    }
}

#[starlark_value(type = "Context")]
impl<'v> StarlarkValue<'v> for Context {}

impl<'v> AllocValue<'v> for Context {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}
