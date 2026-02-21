use crate::models::context::Context;
use anyhow::Context as _;
use starlark::eval::Evaluator;
use starlark::values::{Value, ValueLike};

pub fn get_context<'v, 'a, 'e>(eval: &Evaluator<'v, 'a, 'e>) -> anyhow::Result<&'v Context> {
    eval.module()
        .extra_value()
        .context("Context not found in module extra")?
        .downcast_ref::<Context>()
        .context("Extra value is not a Context")
}

pub fn extract_function_name(function: Value) -> String {
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
