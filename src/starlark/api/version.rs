use crate::models::version_entry::{VersionEntry, InstallStep, Export, BuildFlag};
use anyhow::Context as _;
use starlark::eval::Evaluator;
use starlark::starlark_module;
use starlark::values::{Value, ValueLike, none::NoneType};
use starlark::any::ProvidesStaticType;
use starlark::environment::Methods;
use starlark::environment::MethodsBuilder;
use starlark::environment::MethodsStatic;
use starlark::values::{
    starlark_value, AllocValue, Heap, StarlarkValue,
};
use allocative::Allocative;
use serde::Serialize;
use std::fmt::{self, Debug, Display};
use std::sync::Arc;
use parking_lot::RwLock;
use crate::starlark::api::utils::get_context;
use starlark::environment::GlobalsBuilder;

#[derive(Debug, ProvidesStaticType, Clone, Allocative, Serialize)]
pub struct VersionBuilder {
    pub pkgname: String,
    pub version: String,
    pub release_date: String,
    pub release_type: String,
    pub stream: String,
    pub pipeline: Vec<InstallStep>,
    pub exports: Vec<Export>,
    pub flags: Vec<BuildFlag>,
}

#[derive(Debug, ProvidesStaticType, Clone, Serialize)]
pub struct StarlarkVersionBuilder {
    #[serde(skip)]
    pub builder: Arc<RwLock<VersionBuilder>>,
}

impl Display for StarlarkVersionBuilder {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let b = self.builder.read();
        write!(f, "VersionBuilder({}:{})", b.pkgname, b.version)
    }
}

impl Allocative for StarlarkVersionBuilder {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut allocative::Visitor<'b>) {
        let _visitor = visitor.enter_self_sized::<Self>();
    }
}

#[starlark_value(type = "VersionBuilder")]
impl<'v> StarlarkValue<'v> for StarlarkVersionBuilder {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(version_builder_methods)
    }
}

impl<'v> AllocValue<'v> for StarlarkVersionBuilder {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark_module]
fn version_builder_methods(builder: &mut MethodsBuilder) {
    fn set_stream(this: Value, name: String) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().stream = name;
        Ok(NoneType)
    }

    fn add_flag(
        this: Value,
        name: String,
        help: String,
        default: Value,
    ) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        let default_value = match default.unpack_bool() {
            Some(b) => b.to_string(),
            None => default.to_value().to_str(),
        };
        this.builder.write().flags.push(BuildFlag {
            name,
            help,
            default_value,
        });
        Ok(NoneType)
    }

    fn flag_value<'v>(this: Value<'v>, name: String, eval: &mut Evaluator<'v, '_, '_>) -> anyhow::Result<Value<'v>> {
        let builder_val = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        let context = get_context(eval)?;
        
        // Find flag definition to get default
        let b = builder_val.builder.read();
        let flag_def = b.flags.iter().find(|f| f.name == name);
        
        let val = context.options.get(&name).cloned()
            .or_else(|| flag_def.map(|f| f.default_value.clone()));

        match val {
            Some(v) => Ok(eval.heap().alloc(v)),
            None => Ok(Value::new_none()),
        }
    }

    fn fetch(
        this: Value, 
        url: String, 
        checksum: Option<String>, 
        filename: Option<String>, 
        name: Option<String>
    ) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().pipeline.push(InstallStep::Fetch { url, checksum, filename, name });
        Ok(NoneType)
    }

    fn extract(
        this: Value, 
        format: Option<String>, 
        name: Option<String>
    ) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().pipeline.push(InstallStep::Extract { format, name });
        Ok(NoneType)
    }

    fn run(
        this: Value, 
        command: String, 
        cwd: Option<String>, 
        name: Option<String>
    ) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().pipeline.push(InstallStep::Run { command, cwd, name });
        Ok(NoneType)
    }

    fn export_link(this: Value, src: String, dest: String) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().exports.push(Export::Link { src, dest });
        Ok(NoneType)
    }

    fn export_env(this: Value, key: String, val: String) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().exports.push(Export::Env { key, val });
        Ok(NoneType)
    }

    fn export_path(this: Value, path: String) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().exports.push(Export::Path(path));
        Ok(NoneType)
    }

    fn register(this: Value, eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;
        let svb = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        let b = svb.builder.read();
        
        context.versions.write().push(VersionEntry {
            pkgname: b.pkgname.clone(),
            version: b.version.clone(),
            release_date: b.release_date.clone(),
            release_type: b.release_type.clone(),
            stream: b.stream.clone(),
            pipeline: b.pipeline.clone(),
            exports: b.exports.clone(),
            flags: b.flags.clone(),
        });
        Ok(NoneType)
    }
}

#[starlark_module]
pub fn register_version_globals(builder: &mut GlobalsBuilder) {
    fn create_version(
        pkgname: String,
        version: String,
        release_date: Option<String>,
        release_type: Option<String>,
    ) -> anyhow::Result<StarlarkVersionBuilder> {
        Ok(StarlarkVersionBuilder {
            builder: Arc::new(RwLock::new(VersionBuilder {
                pkgname,
                version,
                release_date: release_date.unwrap_or_default(),
                release_type: release_type.unwrap_or_else(|| "stable".to_string()),
                stream: String::new(),
                pipeline: Vec::new(),
                exports: Vec::new(),
                flags: Vec::new(),
            }))
        })
    }
}
