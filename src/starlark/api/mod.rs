use crate::models::context::Context;
use crate::models::package_entry::{ManagerEntry, PackageEntry};
use crate::models::version_entry::{ManagerCommand, VersionEntry};
use crate::services::cache::Cache;
use crate::services::downloader::Downloader;
use anyhow::Context as _;
use serde_json_path::JsonPath;
use starlark::environment::GlobalsBuilder;
use starlark::eval::Evaluator;
use starlark::starlark_module;
use starlark::syntax::{AstModule, Dialect};
use starlark::values::dict::{DictRef, Dict};
use starlark::values::list::ListRef;
use starlark::values::{Heap, Value, ValueLike, none::NoneType};
use starlark::collections::SmallMap;
use std::time::Duration;

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
            println!("From Cache: {}", url);
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
            println!("From Cache: {}", url);
            return Ok(cached);
        }

        println!("Fetching: {}", url);
        let content = Downloader::download(&url)?;
        cache.write(&url, &content)?;
        Ok(content)
    }

    fn json_parse<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let ast = AstModule::parse("internal", "json".to_string(), &Dialect::Extended)
            .map_err(|e| anyhow::anyhow!("{}", e))?;
        let json_val = eval
            .eval_statements(ast)
            .map_err(|e| anyhow::anyhow!("Failed to retrieve json module: {}", e))?;

        let decode = json_val
            .get_attr("decode", eval.heap())
            .map_err(|e| anyhow::anyhow!("{}", e))?
            .context("json.decode not found")?;

        let content_val = eval.heap().alloc(content);

        eval.eval_function(decode, &[content_val], &[])
            .map_err(|e| anyhow::anyhow!("{}", e))
    }

    fn toml_parse<'v>(
        content: String,
        eval: &mut Evaluator<'v, '_, '_>,
    ) -> anyhow::Result<Value<'v>> {
        let json_value: serde_json::Value = toml::from_str(&content)
            .map_err(|e| anyhow::anyhow!("TOML parse error: {}", e))?;
        Ok(serde_to_starlark(json_value, eval.heap()))
    }

    fn json_dump(data: Value, query: Option<String>) -> anyhow::Result<NoneType> {
        let json_val = starlark_to_serde(data)?;

        if let Some(q) = query {
            let path =
                JsonPath::parse(&q).map_err(|e| anyhow::anyhow!("JSONPath parse error: {}", e))?;
            let node = path.query(&json_val);
            println!("{}", serde_json::to_string_pretty(&node)?);
        } else {
            println!("{}", serde_json::to_string_pretty(&json_val)?);
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

fn starlark_to_serde(val: Value) -> anyhow::Result<serde_json::Value> {
    if val.is_none() {
        Ok(serde_json::Value::Null)
    } else if let Some(b) = val.unpack_bool() {
        Ok(serde_json::Value::Bool(b))
    } else if let Some(i) = val.unpack_i32() {
        Ok(serde_json::Value::Number(i.into()))
    } else if let Some(s) = val.unpack_str() {
        Ok(serde_json::Value::String(s.to_string()))
    } else if let Some(list) = ListRef::from_value(val) {
        let mut arr = Vec::new();
        for v in list.content() {
            arr.push(starlark_to_serde(*v)?);
        }
        Ok(serde_json::Value::Array(arr))
    } else if let Some(dict) = DictRef::from_value(val) {
        let mut obj = serde_json::Map::new();
        for (k, v) in dict.iter_hashed() {
            let key_str = k.key().to_str();
            obj.insert(key_str, starlark_to_serde(v)?);
        }
        Ok(serde_json::Value::Object(obj))
    } else {
        Ok(serde_json::Value::String(val.to_str()))
    }
}

fn serde_to_starlark<'v>(val: serde_json::Value, heap: &'v Heap) -> Value<'v> {
    match val {
        serde_json::Value::Null => Value::new_none(),
        serde_json::Value::Bool(b) => Value::new_bool(b),
        serde_json::Value::Number(n) => {
            if let Some(i) = n.as_i64() {
                heap.alloc(i as i32) // Starlark i32
            } else if let Some(f) = n.as_f64() {
                heap.alloc(f)
            } else {
                heap.alloc(n.to_string())
            }
        }
        serde_json::Value::String(s) => heap.alloc(s),
        serde_json::Value::Array(arr) => {
            let mut list = Vec::new();
            for v in arr {
                list.push(serde_to_starlark(v, heap));
            }
            heap.alloc(list)
        }
        serde_json::Value::Object(obj) => {
            let mut dict = SmallMap::with_capacity(obj.len());
            for (k, v) in obj {
                dict.insert_hashed(heap.alloc(k).get_hashed().unwrap(), serde_to_starlark(v, heap));
            }
            heap.alloc(Dict::new(dict))
        }
    }
}
