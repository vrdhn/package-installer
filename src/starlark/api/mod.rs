use crate::models::context::Context;
use crate::models::package_entry::{ManagerEntry, PackageEntry};
use crate::models::version_entry::{VersionEntry, InstallStep, Export};
use crate::services::cache::Cache;
use crate::services::downloader::Downloader;
use anyhow::Context as _;
use starlark::environment::GlobalsBuilder;
use starlark::eval::Evaluator;
use starlark::starlark_module;
use starlark::values::{Value, ValueLike, none::NoneType};
use std::time::Duration;
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

mod xml;
mod html;
mod data;

#[derive(Debug, ProvidesStaticType, Clone, Allocative, Serialize)]
pub struct VersionBuilder {
    pub pkgname: String,
    pub version: String,
    pub release_date: String,
    pub release_type: String,
    pub pipeline: Vec<InstallStep>,
    pub exports: Vec<Export>,
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
    fn fetch(this: Value, url: String, checksum: Option<String>, filename: Option<String>) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().pipeline.push(InstallStep::Fetch { url, checksum, filename });
        Ok(NoneType)
    }

    fn extract(this: Value, format: Option<String>) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().pipeline.push(InstallStep::Extract { format });
        Ok(NoneType)
    }

    fn run(this: Value, command: String, cwd: Option<String>) -> anyhow::Result<NoneType> {
        let this = this.downcast_ref::<StarlarkVersionBuilder>().context("not a VersionBuilder")?;
        this.builder.write().pipeline.push(InstallStep::Run { command, cwd });
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
}

#[starlark_module]
pub fn register_api(builder: &mut GlobalsBuilder) {
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
                pipeline: Vec::new(),
                exports: Vec::new(),
            }))
        })
    }

    fn add_version(
        builder: Value,
        eval: &mut Evaluator<'_, '_, '_>,
    ) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;
        let svb = builder.downcast_ref::<StarlarkVersionBuilder>().context("builder must be a VersionBuilder")?;
        let b = svb.builder.read();
        
        context.versions.write().push(VersionEntry {
            pkgname: b.pkgname.clone(),
            version: b.version.clone(),
            release_date: b.release_date.clone(),
            release_type: b.release_type.clone(),
            pipeline: b.pipeline.clone(),
            exports: b.exports.clone(),
        });
        Ok(NoneType)
    }

    fn get_os(eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context = get_context(eval)?;
        Ok(context.os.clone())
    }

    fn get_arch(eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context = get_context(eval)?;
        Ok(context.arch.clone())
    }

    fn add_package<'v>(
        name: String,
        function: Value<'v>,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;
        let function_name = extract_function_name(function);

        context.packages.write().push(PackageEntry {
            name,
            function_name,
            filename: context.filename.clone(),
        });

        Ok(NoneType)
    }

    fn add_manager<'v>(
        name: String,
        function: Value<'v>,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;
        let function_name = extract_function_name(function);

        context.managers.write().push(ManagerEntry {
            name,
            function_name,
            filename: context.filename.clone(),
        });

        Ok(NoneType)
    }

    fn download(url: String, eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context = get_context(eval)?;
        let cache = Cache::new(context.meta_dir.clone(), Duration::from_secs(3600)); // 1 hour TTL

        if let Some(cached) = cache.read(&url)? {
            log::debug!("[{}] cache hit: {}", context.display_name(), url);
            return Ok(cached);
        }

        let lock = context
            .state
            .download_locks
            .entry(url.clone())
            .or_insert_with(|| std::sync::Arc::new(parking_lot::Mutex::new(())))
            .clone();

        let _guard = lock.lock();

        if let Some(cached) = cache.read(&url)? {
            log::debug!("[{}] cache hit: {}", context.display_name(), url);
            return Ok(cached);
        }

        log::info!("[{}] fetching: {}", context.display_name(), url);
        let content = Downloader::download(&url)?;
        cache.write(&url, &content)?;
        Ok(content)
    }

    fn parse_json<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let context = get_context(eval)?;
        let json_value: serde_json::Value = serde_json::from_str(&content)
            .map_err(|e| anyhow::anyhow!("[{}] JSON parse error: {}", context.display_name(), e))?;
        Ok(eval.heap().alloc(data::DataDocument { value: json_value }))
    }

    fn parse_toml<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let context = get_context(eval)?;
        let json_value: serde_json::Value = toml::from_str(&content)
            .map_err(|e| anyhow::anyhow!("[{}] TOML parse error: {}", context.display_name(), e))?;
        Ok(eval.heap().alloc(data::DataDocument { value: json_value }))
    }

    fn parse_xml<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let element = xmltree::Element::parse(content.as_bytes())
            .map_err(|e| anyhow::anyhow!("XML parse error: {}", e))?;
        Ok(eval.heap().alloc(xml::XmlDocument { root: element }))
    }

    fn parse_html<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let document = std::sync::Arc::new(std::sync::Mutex::new(scraper::Html::parse_document(&content)));
        let doc_obj = html::HtmlDocument { doc: document };
        Ok(eval.heap().alloc(doc_obj))
    }

    fn json_dump(data: Value, query: Option<String>, eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;
        let json_val = data::starlark_to_serde(data)?;

        if let Some(q) = query {
            let path =
                serde_json_path::JsonPath::parse(&q).map_err(|e| anyhow::anyhow!("[{}] JSONPath parse error: {}", context.display_name(), e))?;
            let node = path.query(&json_val);
            log::info!("[{}] {}", context.display_name(), serde_json::to_string_pretty(&node)?);
        } else {
            log::info!("[{}] {}", context.display_name(), serde_json::to_string_pretty(&json_val)?);
        }

        Ok(NoneType)
    }
}

fn get_context<'v, 'a, 'e>(eval: &Evaluator<'v, 'a, 'e>) -> anyhow::Result<&'v Context> {
    eval.module()
        .extra_value()
        .context("Context not found in module extra")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")
}

fn extract_function_name(function: Value) -> String {
    let repr = function.to_value().to_str();
    let name = if let Some(s) = repr.strip_prefix("<function ") {
        s.strip_suffix(">").unwrap_or(s)
    } else {
        &repr
    };

    name.rfind('.')
        .map(|idx| &name[idx + 1..])
        .unwrap_or(name)
        .to_string()
}
