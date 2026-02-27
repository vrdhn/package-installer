use crate::models::package_entry::{ManagerEntry, PackageEntry};
use crate::services::cache::Cache;
use crate::services::downloader::Downloader;
use starlark::eval::Evaluator;
use starlark::values::{Value, none::NoneType};
use std::time::Duration;
use crate::starlark::api::data;
use crate::starlark::api::xml;
use crate::starlark::api::html;
use crate::starlark::api::utils::{get_context, extract_function_name};
use starlark::environment::GlobalsBuilder;
use starlark::starlark_module;

pub fn register_stdlib(builder: &mut GlobalsBuilder) {
    register_stdlib_internal(builder);
}

fn match_re_logic<'v>(
    pattern: &str,
    text: &str,
    eval: &mut Evaluator<'v, '_, '_>,
) -> anyhow::Result<Value<'v>> {
    let re = regex::Regex::new(pattern).map_err(|e| anyhow::anyhow!("Regex error: {}", e))?;

    if let Some(caps) = re.captures(text) {
        let mut res = Vec::with_capacity(caps.len());
        res.push(eval.heap().alloc(true));
        for i in 1..caps.len() {
            res.push(eval.heap().alloc(caps.get(i).map(|m| m.as_str()).unwrap_or("")));
        }
        Ok(eval.heap().alloc(res))
    } else {
        let mut res = Vec::with_capacity(re.captures_len());
        res.push(eval.heap().alloc(false));
        for _ in 1..re.captures_len() {
            res.push(eval.heap().alloc(""));
        }
        Ok(eval.heap().alloc(res))
    }
}

#[starlark_module]
fn register_stdlib_internal(builder: &mut GlobalsBuilder) {
    fn extract<'v>(
        pattern: String,
        text: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        match_re_logic(&pattern, &text, eval)
    }

    fn re_match<'v>(
        pattern: String,
        text: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        match_re_logic(&pattern, &text, eval)
    }

    fn get_os(eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context = get_context(eval)?;
        Ok(context.os.to_string())
    }

    fn get_arch(eval: &mut Evaluator<'_, '_, '_>) -> anyhow::Result<String> {
        let context = get_context(eval)?;
        Ok(context.arch.to_string())
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

        if !context.force {
            if let Some(cached) = cache.read(&url)? {
                log::debug!("[{}] cache hit: {}", context.display_name(), url);
                return Ok(cached);
            }
        }

        // Acquire or create a per-URL download lock to avoid redundant concurrent requests.
        // We drop the DashMap entry lock quickly by cloning the Arc<Mutex<()>>.
        let lock = context
            .state
            .download_locks
            .entry(url.clone())
            .or_insert_with(|| std::sync::Arc::new(parking_lot::Mutex::new(())))
            .clone();

        // Hold the Mutex during the download process to ensure only one thread performs it.
        let _guard = lock.lock();

        if !context.force {
            if let Some(cached) = cache.read(&url)? {
                log::debug!("[{}] cache hit: {}", context.display_name(), url);
                return Ok(cached);
            }
        }

        log::info!("[{}] fetching: {}", context.display_name(), url);
        let content = match Downloader::download(&url) {
            Ok(c) => c,
            Err(e) => {
                log::warn!("[{}] download failed for {}: {}", context.display_name(), url, e);
                return Ok(String::new());
            }
        };
        cache.write(&url, &content)?;
        Ok(content)
    }

    fn parse_json<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let context = get_context(eval)?;
        if content.is_empty() {
            return Ok(eval.heap().alloc(data::DataDocument { value: serde_json::Value::Object(serde_json::Map::new()) }));
        }
        let json_value: serde_json::Value = serde_json::from_str(&content)
            .map_err(|e| anyhow::anyhow!("[{}] JSON parse error: {}", context.display_name(), e))?;
        Ok(eval.heap().alloc(data::DataDocument { value: json_value }))
    }

    fn parse_toml<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let context = get_context(eval)?;
        if content.is_empty() {
            return Ok(eval.heap().alloc(data::DataDocument { value: serde_json::Value::Object(serde_json::Map::new()) }));
        }
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
