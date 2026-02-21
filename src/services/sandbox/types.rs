use std::path::PathBuf;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum BindType {
    Bind,
    BindTry,
    DevBind,
    DevBindTry,
    RoBind,
    RoBindTry,
    Proc,
    Dev,
    Tmpfs,
    Dir,
}

impl BindType {
    pub fn as_str(&self) -> &'static str {
        match self {
            BindType::Bind => "--bind",
            BindType::BindTry => "--bind-try",
            BindType::DevBind => "--dev-bind",
            BindType::DevBindTry => "--dev-bind-try",
            BindType::RoBind => "--ro-bind",
            BindType::RoBindTry => "--ro-bind-try",
            BindType::Proc => "--proc",
            BindType::Dev => "--dev",
            BindType::Tmpfs => "--tmpfs",
            BindType::Dir => "--dir",
        }
    }
}

#[derive(Debug, Clone)]
pub struct BindPair {
    pub cave_target: PathBuf,
    pub host_source: Option<PathBuf>,
    pub bind_type: BindType,
}
