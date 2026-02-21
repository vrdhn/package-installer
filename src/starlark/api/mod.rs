use starlark::environment::GlobalsBuilder;

pub mod data;
pub mod html;
pub mod stdlib;
pub mod version;
pub mod xml;
pub mod utils;

pub fn register_api(builder: &mut GlobalsBuilder) {
    stdlib::register_stdlib(builder);
    version::register_version_globals(builder);
}
