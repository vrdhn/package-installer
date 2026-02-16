use crate::models::context::Context;
use crate::models::package_entry::{ManagerEntry, PackageEntry};
use crate::models::version_entry::{ManagerCommand, VersionEntry};
use crate::services::cache::Cache;
use crate::services::downloader::Downloader;
use anyhow::Context as _;
use starlark::environment::GlobalsBuilder;
use starlark::eval::Evaluator;
use starlark::starlark_module;
use starlark::values::dict::DictRef;
use starlark::values::{Value, ValueLike, none::NoneType};
use std::time::Duration;

mod xml;
mod html;
mod data;

#[starlark_module]
pub fn register_api(builder: &mut GlobalsBuilder) {
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

        // Handle concurrent downloads by locking on the URL
        let lock = context
            .state
            .download_locks
            .entry(url.clone())
            .or_insert_with(|| std::sync::Arc::new(parking_lot::Mutex::new(())))
            .clone();

        let _guard = lock.lock();

        // Check cache again after acquiring lock to see if another thread finished it
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

    fn add_version(
        pkgname: String,
        version: String,
        release_date: String,
        release_type: String,
        url: String,
        filename: String,
        checksum: String,
        checksum_url: String,
        filemap: Option<Value>,
        env: Option<Value>,
        manager_command: Option<String>,
        eval: &mut Evaluator<'_, '_, '_>,
    ) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;

        if !is_valid_release_type(&release_type) {
            anyhow::bail!(
                "[{}] invalid release_type: '{}'. Must be 'lts', 'stable', 'testing', 'unstable' or match pattern 'major[.minor[.patch]][-suffix]'",
                context.display_name(),
                release_type
            );
        }

        let cmd = match manager_command {
            Some(c) => ManagerCommand::Custom(c),
            None => ManagerCommand::Auto,
        };

        let mut parsed_filemap = std::collections::HashMap::new();
        if let Some(s) = filemap {
            let dict = DictRef::from_value(s).context("filemap must be a dictionary")?;
            for (k, v) in dict.iter_hashed() {
                let key = k.key().unpack_str().context("filemap key must be a string")?;
                let val = v.unpack_str().context("filemap value must be a string")?;
                parsed_filemap.insert(key.to_string(), val.to_string());
            }
        };

        let mut parsed_env = std::collections::HashMap::new();
        if let Some(e) = env {
            let dict = DictRef::from_value(e).context("env must be a dictionary")?;
            for (k, v) in dict.iter_hashed() {
                let key = k.key().unpack_str().context("env key must be a string")?;
                let val = v.unpack_str().context("env value must be a string")?;
                parsed_env.insert(key.to_string(), val.to_string());
            }
        };

        context.versions.write().push(VersionEntry {
            pkgname,
            version,
            release_date,
            release_type,
            url,
            filename,
            checksum,
            checksum_url,
            filemap: parsed_filemap,
            env: parsed_env,
            manager_command: cmd,
        });
        Ok(NoneType)
    }
}

fn is_valid_release_type(rt: &str) -> bool {
    match rt {
        "lts" | "stable" | "testing" | "unstable" => true,
        _ => {
            let (version_part, _suffix) = match rt.find('-') {
                Some(idx) => (&rt[..idx], Some(&rt[idx + 1..])),
                None => (rt, None),
            };

            let parts: Vec<&str> = version_part.split('.').collect();
            if parts.is_empty() || parts.len() > 3 {
                return false;
            }

            for part in parts {
                if part.is_empty() || !part.chars().all(|c| c.is_ascii_digit()) {
                    return false;
                }
            }
            true
        }
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
