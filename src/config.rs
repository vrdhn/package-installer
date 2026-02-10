use starlark::values::{starlark_value, StarlarkValue, Value, ValueLike, AllocValue, Heap, none::NoneType};
use starlark::any::ProvidesStaticType;
use starlark::starlark_module;
use starlark::environment::GlobalsBuilder;
use starlark::eval::Evaluator;
use std::env;
use std::fmt::{self, Display, Debug};
use parking_lot::RwLock;
use allocative::{Allocative, Visitor, Key};
use serde::Serialize;
use anyhow::Context as _;

#[derive(Debug, Clone, Allocative, Serialize)]
pub struct PackageEntry {
    pub regexp: String,
    pub function_name: String,
    pub filename: String,
}

#[derive(Debug, ProvidesStaticType, Serialize)]
pub struct Context {
    pub os: String,
    pub arch: String,
    pub filename: String,
    pub packages: RwLock<Vec<PackageEntry>>,
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

#[starlark_module]
pub fn starlark_functions(builder: &mut GlobalsBuilder) {
    fn get_os(eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context_value = eval.module().extra_value().context("Context not found in module extra")?;
        let context = context_value.downcast_ref::<Context>().context("Extra value is not a Context")?;
        Ok(context.os.clone())
    }

    fn get_arch(eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context_value = eval.module().extra_value().context("Context not found in module extra")?;
        let context = context_value.downcast_ref::<Context>().context("Extra value is not a Context")?;
        Ok(context.arch.clone())
    }

    fn add_package<'v>(regexp: String, function: Value<'v>, eval: &mut Evaluator<'v, '_, '_>) -> anyhow::Result<NoneType> {
        let context_value = eval.module().extra_value().context("Context not found in module extra")?;
        let context = context_value.downcast_ref::<Context>().context("Extra value is not a Context")?;
        
        let function_repr = function.to_value().to_str();
        let mut name = if let Some(s) = function_repr.strip_prefix("<function ") {
            s.strip_suffix(">").unwrap_or(s).to_string()
        } else {
            function_repr.to_string()
        };

        if let Some(last_dot) = name.rfind('.') {
            name = name[last_dot + 1..].to_string();
        }

        context.packages.write().push(PackageEntry {
            regexp,
            function_name: name,
            filename: context.filename.clone(),
        });
        
        Ok(NoneType)
    }
}
