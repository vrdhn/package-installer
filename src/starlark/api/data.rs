use starlark::any::ProvidesStaticType;
use starlark::environment::Methods;
use starlark::environment::MethodsBuilder;
use starlark::environment::MethodsStatic;
use starlark::values::{
    starlark_value, AllocValue, Heap, StarlarkValue, Value, ValueLike,
};
use starlark::values::list::ListRef;
use starlark::values::dict::DictRef;
use std::fmt::{self, Display};
use allocative::{Allocative, Visitor};
use serde::Serialize;
use anyhow::Context;

#[derive(Debug, ProvidesStaticType, Clone, Serialize)]
pub struct DataDocument {
    pub value: serde_json::Value,
}

impl Display for DataDocument {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "DataDocument")
    }
}

impl Allocative for DataDocument {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let visitor = visitor.enter_self_sized::<Self>();
        visitor.exit();
    }
}

#[starlark_value(type = "DataDocument")]
impl<'v> StarlarkValue<'v> for DataDocument {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(data_document_methods)
    }
}

impl<'v> AllocValue<'v> for DataDocument {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark::starlark_module]
fn data_document_methods(builder: &mut MethodsBuilder) {
    #[starlark(attribute)]
    fn root<'v>(this: Value<'v>, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<DataDocument>().context("not a DataDocument")?;
        Ok(heap.alloc(DataNode { value: this.value.clone() }))
    }

    fn dump(this: Value<'v>) -> anyhow::Result<String> {
        let this = this.downcast_ref::<DataDocument>().context("not a DataDocument")?;
        Ok(serde_json::to_string_pretty(&this.value)?)
    }
}

#[derive(Debug, ProvidesStaticType, Clone, Serialize)]
pub struct DataNode {
    pub value: serde_json::Value,
}

impl Allocative for DataNode {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let visitor = visitor.enter_self_sized::<Self>();
        visitor.exit();
    }
}

impl Display for DataNode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "DataNode({})", self.value)
    }
}

#[starlark_value(type = "DataNode")]
impl<'v> StarlarkValue<'v> for DataNode {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(data_node_methods)
    }

    fn at(&self, index: Value<'v>, heap: &'v Heap) -> starlark::Result<Value<'v>> {
        match &self.value {
            serde_json::Value::Array(arr) => {
                let i = index.unpack_i32().context("index must be an integer").map_err(|e| starlark::Error::new_other(e))?;
                if i < 0 || i as usize >= arr.len() {
                    return Err(starlark::Error::new_other(anyhow::anyhow!("index out of bounds")));
                }
                Ok(serde_to_starlark(arr[i as usize].clone(), heap))
            }
            serde_json::Value::Object(obj) => {
                let key = index.unpack_str().context("index must be a string").map_err(|e| starlark::Error::new_other(e))?;
                if let Some(val) = obj.get(key) {
                    Ok(serde_to_starlark(val.clone(), heap))
                } else {
                    Err(starlark::Error::new_other(anyhow::anyhow!("key not found: {}", key)))
                }
            }
            _ => Err(starlark::Error::new_other(anyhow::anyhow!("object is not indexable"))),
        }
    }

    fn length(&self) -> starlark::Result<i32> {
        match &self.value {
            serde_json::Value::Array(arr) => Ok(arr.len() as i32),
            serde_json::Value::Object(obj) => Ok(obj.len() as i32),
            serde_json::Value::String(s) => Ok(s.len() as i32),
            _ => Ok(0),
        }
    }

    fn iterate_collect(&self, heap: &'v Heap) -> starlark::Result<Vec<Value<'v>>> {
        match &self.value {
            serde_json::Value::Array(arr) => {
                Ok(arr.iter().map(|v| serde_to_starlark(v.clone(), heap)).collect())
            }
            serde_json::Value::Object(obj) => {
                Ok(obj.keys().map(|k| heap.alloc(k.clone())).collect())
            }
            _ => Err(starlark::Error::new_other(anyhow::anyhow!("object is not iterable"))),
        }
    }
}

impl<'v> AllocValue<'v> for DataNode {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark::starlark_module]
fn data_node_methods(builder: &mut MethodsBuilder) {
    fn get<'v>(
        this: Value<'v>,
        key: String,
        default: Option<Value<'v>>,
        heap: &'v Heap,
    ) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<DataNode>().context("not a DataNode")?;
        let default_val = default.unwrap_or_else(Value::new_none);
        match &this.value {
            serde_json::Value::Object(obj) => {
                if let Some(val) = obj.get(&key) {
                    Ok(serde_to_starlark(val.clone(), heap))
                } else {
                    Ok(default_val)
                }
            }
            _ => Ok(default_val),
        }
    }

    fn select<'v>(this: Value<'v>, query: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<DataNode>().context("not a DataNode")?;
        let path = serde_json_path::JsonPath::parse(&query).map_err(|e| anyhow::anyhow!("JSONPath parse error: {}", e))?;
        let node = path.query(&this.value);
        
        let mut result = Vec::new();
        for v in node {
             result.push(serde_to_starlark(v.clone(), heap));
        }
        Ok(heap.alloc(result))
    }

    fn select_one<'v>(this: Value<'v>, query: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<DataNode>().context("not a DataNode")?;
        let path = serde_json_path::JsonPath::parse(&query).map_err(|e| anyhow::anyhow!("JSONPath parse error: {}", e))?;
        let node = path.query(&this.value);
        
        if let Some(v) = node.first() {
            Ok(serde_to_starlark(v.clone(), heap))
        } else {
            Ok(Value::new_none())
        }
    }

    fn attribute<'v>(this: Value<'v>, name: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<DataNode>().context("not a DataNode")?;
        match &this.value {
            serde_json::Value::Object(obj) => {
                if let Some(val) = obj.get(&name) {
                    match val {
                        serde_json::Value::String(s) => Ok(heap.alloc(s.clone())),
                        _ => Ok(heap.alloc(val.to_string())),
                    }
                } else {
                    Ok(Value::new_none())
                }
            }
            _ => Ok(Value::new_none()),
        }
    }

    fn text(this: Value) -> anyhow::Result<String> {
        let this = this.downcast_ref::<DataNode>().context("not a DataNode")?;
        match &this.value {
            serde_json::Value::String(s) => Ok(s.clone()),
            serde_json::Value::Number(n) => Ok(n.to_string()),
            serde_json::Value::Bool(b) => Ok(b.to_string()),
            _ => Ok(this.value.to_string()),
        }
    }

    fn dump(this: Value<'v>) -> anyhow::Result<String> {
        let this = this.downcast_ref::<DataNode>().context("not a DataNode")?;
        Ok(serde_json::to_string_pretty(&this.value)?)
    }
}

pub fn starlark_to_serde(val: Value) -> anyhow::Result<serde_json::Value> {
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
    } else if let Some(node) = val.downcast_ref::<DataNode>() {
        Ok(node.value.clone())
    } else if let Some(doc) = val.downcast_ref::<DataDocument>() {
        Ok(doc.value.clone())
    } else {
        Ok(serde_json::Value::String(val.to_str()))
    }
}

pub fn serde_to_starlark<'v>(val: serde_json::Value, heap: &'v Heap) -> Value<'v> {
    match val {
        serde_json::Value::Null => Value::new_none(),
        serde_json::Value::Bool(b) => Value::new_bool(b),
        serde_json::Value::Number(n) => {
            if let Some(i) = n.as_i64() {
                heap.alloc(i as i32)
            } else if let Some(f) = n.as_f64() {
                heap.alloc(f)
            } else {
                heap.alloc(n.to_string())
            }
        }
        serde_json::Value::String(s) => heap.alloc(s),
        serde_json::Value::Array(arr) => {
            heap.alloc(DataNode { value: serde_json::Value::Array(arr) })
        }
        serde_json::Value::Object(obj) => {
            heap.alloc(DataNode { value: serde_json::Value::Object(obj) })
        }
    }
}
