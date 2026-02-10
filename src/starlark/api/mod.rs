use starlark::values::{Value, ValueLike, none::NoneType};
use starlark::starlark_module;
use starlark::environment::GlobalsBuilder;
use starlark::eval::Evaluator;
use anyhow::Context as _;
use crate::models::context::Context;
use crate::models::package_entry::PackageEntry;

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

    fn add_package<'v>(regexp: String, function: Value<'v>, eval: &mut Evaluator<'v, '_, '_>) -> anyhow::Result<NoneType> {
        let context = get_context(eval)?;
        let name = extract_function_name(function);
        
        context.packages.write().push(PackageEntry {
            regexp,
            function_name: name,
            filename: context.filename.clone(),
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
