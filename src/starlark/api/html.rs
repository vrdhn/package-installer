use starlark::any::ProvidesStaticType;
use starlark::environment::Methods;
use starlark::environment::MethodsBuilder;
use starlark::environment::MethodsStatic;
use starlark::values::{
    starlark_value, AllocValue, Heap, StarlarkValue, Value, ValueLike,
};
use std::fmt::{self, Display};
use scraper::{Html, ElementRef};
use allocative::{Allocative, Visitor};
use serde::Serialize;
use anyhow::Context;
use std::sync::{Arc, Mutex};
use ego_tree::NodeId;

#[derive(Debug, ProvidesStaticType, Clone)]
pub struct HtmlDocument {
    pub doc: Arc<Mutex<Html>>,
}

impl Serialize for HtmlDocument {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str("HtmlDocument")
    }
}

impl Display for HtmlDocument {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "HtmlDocument")
    }
}

impl Allocative for HtmlDocument {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let visitor = visitor.enter_self_sized::<Self>();
        visitor.exit();
    }
}

#[starlark_value(type = "HtmlDocument")]
impl<'v> StarlarkValue<'v> for HtmlDocument {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(html_document_methods)
    }
}

impl<'v> AllocValue<'v> for HtmlDocument {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark::starlark_module]
fn html_document_methods(builder: &mut MethodsBuilder) {
    #[starlark(attribute)]
    fn root<'v>(this: Value<'v>, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<HtmlDocument>().context("not an HtmlDocument")?;
        let guard = this.doc.lock().unwrap();
        let node_id = guard.tree.root().id();
        Ok(heap.alloc(HtmlNode {
            doc: this.doc.clone(),
            node_id,
        }))
    }
}

#[derive(Debug, ProvidesStaticType, Clone)]
pub struct HtmlNode {
    pub doc: Arc<Mutex<Html>>,
    pub node_id: NodeId,
}

impl Serialize for HtmlNode {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let guard = self.doc.lock().unwrap();
        if let Some(node) = guard.tree.get(self.node_id) {
             if let Some(element) = ElementRef::wrap(node) {
                 return serializer.serialize_str(&element.html());
             }
        }
        serializer.serialize_none()
    }
}

impl Allocative for HtmlNode {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let visitor = visitor.enter_self_sized::<Self>();
        visitor.exit();
    }
}

impl Display for HtmlNode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let guard = self.doc.lock().unwrap();
        if let Some(node) = guard.tree.get(self.node_id) {
            if let Some(element) = ElementRef::wrap(node) {
                return write!(f, "<{} ...>", element.value().name());
            }
        }
        write!(f, "<unknown>")
    }
}

#[starlark_value(type = "HtmlNode")]
impl<'v> StarlarkValue<'v> for HtmlNode {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(html_node_methods)
    }
}

impl<'v> AllocValue<'v> for HtmlNode {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark::starlark_module]
fn html_node_methods(builder: &mut MethodsBuilder) {
    fn select_one<'v>(this: Value<'v>, selector: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<HtmlNode>().context("not an HtmlNode")?;
        let selector = scraper::Selector::parse(&selector).map_err(|e| anyhow::anyhow!("CSS selector parse error: {:?}", e))?;
        
        let guard = this.doc.lock().unwrap();
        
        if this.node_id == guard.tree.root().id() {
            if let Some(el) = guard.select(&selector).next() {
                return Ok(heap.alloc(HtmlNode {
                    doc: this.doc.clone(),
                    node_id: el.id(),
                }));
            }
        } else {
            let node = guard.tree.get(this.node_id).context("node not found")?;
            if let Some(element) = ElementRef::wrap(node) {
                if let Some(el) = element.select(&selector).next() {
                    return Ok(heap.alloc(HtmlNode {
                        doc: this.doc.clone(),
                        node_id: el.id(),
                    }));
                }
            }
        }
        Ok(Value::new_none())
    }

    fn select<'v>(this: Value<'v>, selector: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<HtmlNode>().context("not an HtmlNode")?;
        let selector = scraper::Selector::parse(&selector).map_err(|e| anyhow::anyhow!("CSS selector parse error: {:?}", e))?;
        
        let guard = this.doc.lock().unwrap();
        let mut result = Vec::new();

        if this.node_id == guard.tree.root().id() {
             for el in guard.select(&selector) {
                result.push(heap.alloc(HtmlNode {
                    doc: this.doc.clone(),
                    node_id: el.id(),
                }));
             }
        } else {
            let node = guard.tree.get(this.node_id).context("node not found")?;
            if let Some(element) = ElementRef::wrap(node) {
                for el in element.select(&selector) {
                    result.push(heap.alloc(HtmlNode {
                        doc: this.doc.clone(),
                        node_id: el.id(),
                    }));
                }
            }
        }
        Ok(heap.alloc(result))
    }

    fn attribute<'v>(this: Value<'v>, name: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<HtmlNode>().context("not an HtmlNode")?;
        let guard = this.doc.lock().unwrap();
        let node = guard.tree.get(this.node_id).context("node not found")?;
        let element = ElementRef::wrap(node).context("not an element")?;
        if let Some(val) = element.value().attr(&name) {
            Ok(heap.alloc(val.to_string()))
        } else {
            Ok(Value::new_none())
        }
    }

    fn text(this: Value) -> anyhow::Result<String> {
        let this = this.downcast_ref::<HtmlNode>().context("not an HtmlNode")?;
        let guard = this.doc.lock().unwrap();
        let node = guard.tree.get(this.node_id).context("node not found")?;
        let element = ElementRef::wrap(node).context("not an element")?;
        let text = element.text().collect::<Vec<_>>().join("");
        Ok(text)
    }

    #[starlark(attribute)]
    fn tag(this: Value) -> anyhow::Result<String> {
        let this = this.downcast_ref::<HtmlNode>().context("not an HtmlNode")?;
        let guard = this.doc.lock().unwrap();
        let node = guard.tree.get(this.node_id).context("node not found")?;
        let element = ElementRef::wrap(node).context("not an element")?;
        Ok(element.value().name().to_string())
    }
}
