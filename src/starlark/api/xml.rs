use starlark::any::ProvidesStaticType;
use starlark::environment::Methods;
use starlark::environment::MethodsBuilder;
use starlark::environment::MethodsStatic;
use starlark::values::{
    starlark_value, AllocValue, Heap, StarlarkValue, Value, ValueLike,
};
use std::fmt::{self, Display};
use xmltree::Element;
use allocative::{Allocative, Visitor};
use serde::Serialize;
use anyhow::Context;

#[derive(Debug, ProvidesStaticType, Clone)]
pub struct XmlDocument {
    pub root: Element,
}

impl Serialize for XmlDocument {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str("XmlDocument")
    }
}

impl Display for XmlDocument {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "XmlDocument")
    }
}

impl Allocative for XmlDocument {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let mut visitor = visitor.enter_self_sized::<Self>();
        visitor.visit_simple_sized::<Element>();
        visitor.exit();
    }
}

#[starlark_value(type = "XmlDocument")]
impl<'v> StarlarkValue<'v> for XmlDocument {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(xml_document_methods)
    }
}

impl<'v> AllocValue<'v> for XmlDocument {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark::starlark_module]
fn xml_document_methods(builder: &mut MethodsBuilder) {
    #[starlark(attribute)]
    fn root<'v>(this: Value<'v>, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<XmlDocument>().context("not an XmlDocument")?;
        Ok(heap.alloc(XmlNode { element: this.root.clone() }))
    }
}

#[derive(Debug, ProvidesStaticType, Clone)]
pub struct XmlNode {
    pub element: Element,
}

impl Serialize for XmlNode {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.element.name)
    }
}

impl Allocative for XmlNode {
    fn visit<'a, 'b: 'a>(&self, visitor: &'a mut Visitor<'b>) {
        let mut visitor = visitor.enter_self_sized::<Self>();
        visitor.visit_simple_sized::<Element>();
        visitor.exit();
    }
}

impl Display for XmlNode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "<{} ...>", self.element.name)
    }
}

#[starlark_value(type = "XmlNode")]
impl<'v> StarlarkValue<'v> for XmlNode {
    fn get_methods() -> Option<&'static Methods> {
        static RES: MethodsStatic = MethodsStatic::new();
        RES.methods(xml_node_methods)
    }
}

impl<'v> AllocValue<'v> for XmlNode {
    fn alloc_value(self, heap: &'v Heap) -> Value<'v> {
        heap.alloc_simple(self)
    }
}

#[starlark::starlark_module]
fn xml_node_methods(builder: &mut MethodsBuilder) {
    fn select_one<'v>(this: Value<'v>, name: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<XmlNode>().context("not an XmlNode")?;
        if let Some(el) = this.element.get_child(name) {
            Ok(heap.alloc(XmlNode { element: el.clone() }))
        } else {
            Ok(Value::new_none())
        }
    }

    fn select<'v>(this: Value<'v>, name: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<XmlNode>().context("not an XmlNode")?;
        let mut result = Vec::new();
        for node in &this.element.children {
            if let xmltree::XMLNode::Element(el) = node {
                if el.name == name {
                    result.push(heap.alloc(XmlNode { element: el.clone() }));
                }
            }
        }
        Ok(heap.alloc(result))
    }

    fn attribute<'v>(this: Value<'v>, name: String, heap: &'v Heap) -> anyhow::Result<Value<'v>> {
        let this = this.downcast_ref::<XmlNode>().context("not an XmlNode")?;
        if let Some(val) = this.element.attributes.get(&name) {
            Ok(heap.alloc(val.clone()))
        } else {
            Ok(Value::new_none())
        }
    }

    fn text(this: Value) -> anyhow::Result<String> {
        let this = this.downcast_ref::<XmlNode>().context("not an XmlNode")?;
        Ok(this.element.get_text().map(|t| t.to_string()).unwrap_or_default())
    }

    #[starlark(attribute)]
    fn tag(this: Value) -> anyhow::Result<String> {
        let this = this.downcast_ref::<XmlNode>().context("not an XmlNode")?;
        Ok(this.element.name.clone())
    }
}
