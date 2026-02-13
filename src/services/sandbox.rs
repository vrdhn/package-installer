use std::collections::BTreeMap;
use std::process::Command;
use std::path::{Path, PathBuf};
use std::os::unix::process::CommandExt;
use anyhow::{Context, Result};

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

pub struct Bubblewrap {
    binds: BTreeMap<PathBuf, BindPair>,
    envs: BTreeMap<String, String>,
    unsets: Vec<String>,
    flags: Vec<String>,
    executable: Option<String>,
    args: Vec<String>,
}

impl Bubblewrap {
    pub fn new() -> Self {
        let mut envs = BTreeMap::new();
        for (key, value) in std::env::vars() {
            envs.insert(key, value);
        }

        Self {
            binds: BTreeMap::new(),
            envs,
            unsets: Vec::new(),
            flags: Vec::new(),
            executable: None,
            args: Vec::new(),
        }
    }

    pub fn add_bind<P: AsRef<Path>>(&mut self, typ: BindType, path: P) {
        let path = path.as_ref().to_path_buf();
        self.binds.insert(path.clone(), BindPair {
            cave_target: path.clone(),
            host_source: Some(path),
            bind_type: typ,
        });
    }

    pub fn add_binds<P: AsRef<Path>>(&mut self, typ: BindType, paths: &[P]) {
        for path in paths {
            self.add_bind(typ, path);
        }
    }

    pub fn add_map_bind<P1: AsRef<Path>, P2: AsRef<Path>>(&mut self, typ: BindType, host_path: P1, cave_path: P2) {
        let host_path = host_path.as_ref().to_path_buf();
        let cave_path = cave_path.as_ref().to_path_buf();
        self.binds.insert(cave_path.clone(), BindPair {
            cave_target: cave_path,
            host_source: Some(host_path),
            bind_type: typ,
        });
    }

    pub fn add_virtual<P: AsRef<Path>>(&mut self, typ: BindType, path: P) {
        let path = path.as_ref().to_path_buf();
        self.binds.insert(path.clone(), BindPair {
            cave_target: path,
            host_source: None,
            bind_type: typ,
        });
    }

    pub fn add_flag(&mut self, flag: &str) {
        self.flags.push(flag.to_string());
    }

    pub fn unset_env(&mut self, name: &str) {
        self.unsets.push(name.to_string());
        self.envs.remove(name);
    }

    pub fn set_env(&mut self, name: &str, value: &str) {
        self.envs.insert(name.to_string(), value.to_string());
    }

    pub fn add_env_first(&mut self, name: &str, entry: &str) {
        let val = self.envs.get(name).cloned().unwrap_or_default();
        let mut parts: Vec<String> = val.split(':').filter(|s| !s.is_empty()).map(|s| s.to_string()).collect();
        if !parts.contains(&entry.to_string()) {
            parts.insert(0, entry.to_string());
        }
        self.envs.insert(name.to_string(), parts.join(":"));
    }

    pub fn set_command(&mut self, executable: &str, args: &[String]) {
        self.executable = Some(executable.to_string());
        self.args = args.to_vec();
    }

    pub fn build_command(&self) -> Command {
        let mut cmd = Command::new("/usr/bin/bwrap");

        for flag in &self.flags {
            cmd.arg(flag);
        }

        for (_, bind) in &self.binds {
            cmd.arg(bind.bind_type.as_str());
            if let Some(ref source) = bind.host_source {
                cmd.arg(source);
            }
            cmd.arg(&bind.cave_target);
        }

        for (key, value) in &self.envs {
            cmd.arg("--setenv").arg(key).arg(value);
        }

        for unset in &self.unsets {
            cmd.arg("--unsetenv").arg(unset);
        }

        if let Some(ref exe) = self.executable {
            cmd.arg("--").arg(exe);
            for arg in &self.args {
                cmd.arg(arg);
            }
        }

        cmd
    }

    pub fn spawn(&self) -> Result<()> {
        let mut cmd = self.build_command();
        let status = cmd.status().context("Failed to spawn bubblewrap process")?;
        if !status.success() {
            return Err(anyhow::anyhow!("Bubblewrap process failed with status: {}", status));
        }
        Ok(())
    }

    pub fn exec(&self) -> Result<()> {
        let mut cmd = self.build_command();
        let err = cmd.exec();
        // If exec returns, it's always an error
        Err(anyhow::Error::from(err).context("Failed to exec into bubblewrap"))
    }
}
