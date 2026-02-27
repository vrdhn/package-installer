use crate::models::config::Config;
use crate::models::cave::Cave;
use crate::services::sandbox::{Bubblewrap, BindType};
use std::env;
use std::path::{Path, PathBuf};
use anyhow::{Context, Result};
use std::collections::HashMap;

pub fn run(config: &Config, variant: Option<String>, command: Vec<String>) {
    if let Err(e) = execute_run(config, variant, command) {
        log::error!("run failed: {}", e);
        std::process::exit(1);
    }
}

/// Options for preparing the sandbox environment.
pub struct SandboxOptions<'a> {
    pub config: &'a Config,
    pub cave: &'a Cave,
    pub variant: Option<&'a str>,
    pub package_envs: HashMap<String, String>,
    pub writable_pilocal: bool,
    pub dependency_dirs: Vec<PathBuf>,
}

/// Prepares the Bubblewrap sandbox with necessary binds and environment variables.
/// Example host_pilocal: "/home/user/.cache/pi/pilocals/my-cave"
/// Example internal_pilocal: "/home/user/.pilocal"
pub fn prepare_sandbox(opts: SandboxOptions) -> Result<Bubblewrap> {
    let settings = opts.cave.get_effective_settings(opts.variant).context("failed to get cave settings")?;
    let mut b = Bubblewrap::new();
    let host_home = opts.config.get_host_home();
    let internal_pilocal = host_home.join(".pilocal");

    bind_system_paths(&mut b);
    bind_virtual_fs(&mut b);
    bind_workspace_and_home(&mut b, opts.cave, &host_home)?;
    bind_pilocal_and_caches(&mut b, opts.config, opts.cave, opts.variant, opts.writable_pilocal, &internal_pilocal)?;
    setup_xdg_runtime(&mut b);
    
    bind_dependencies(&mut b, &opts.dependency_dirs);

    setup_environment(&mut b, opts.config, opts.cave, &host_home, &internal_pilocal);
    apply_custom_envs(&mut b, opts.package_envs, &settings.set, &host_home, &internal_pilocal);

    set_sandbox_hostname(&mut b, opts.config, opts.cave, opts.variant);

    Ok(b)
}

fn bind_dependencies(b: &mut Bubblewrap, dependency_dirs: &[PathBuf]) {
    for dir in dependency_dirs {
        if dir.exists() {
            b.add_bind(BindType::RoBind, dir);
            let bin_dir = dir.join("bin");
            if bin_dir.exists() {
                b.add_env_first("PATH", bin_dir.to_str().unwrap());
            }
        }
    }
}

fn set_sandbox_hostname(b: &mut Bubblewrap, config: &Config, cave: &Cave, variant: Option<&str>) {
    let host_hostname = config.get_hostname();
    let (prefix, suffix) = match host_hostname.find('.') {
        Some(idx) => (&host_hostname[..idx], &host_hostname[idx..]),
        None => (host_hostname.as_str(), ""),
    };

    let cave_hostname = if let Some(v) = variant {
        let v = v.strip_prefix(':').unwrap_or(v);
        format!("{}-{}.{}{}", prefix, cave.name, v, suffix)
    } else {
        format!("{}-{}{}", prefix, cave.name, suffix)
    };
    b.set_hostname(&cave_hostname);
}

fn bind_system_paths(b: &mut Bubblewrap) {
    b.add_flag("--unshare-pid");
    b.add_flag("--unshare-uts");
    b.add_flag("--die-with-parent");
    b.add_bind(BindType::RoBind, "/usr");
    b.add_bind(BindType::RoBind, "/lib");
    if Path::new("/lib64").exists() {
        b.add_bind(BindType::RoBind, "/lib64");
    }
    b.add_bind(BindType::RoBind, "/bin");
    b.add_bind(BindType::RoBind, "/sbin");
    b.add_bind(BindType::RoBind, "/etc");
    b.add_bind(BindType::RoBind, "/sys");
}

fn bind_virtual_fs(b: &mut Bubblewrap) {
    b.add_virtual(BindType::Proc, "/proc");
    b.add_virtual(BindType::Dev, "/dev");
    b.add_virtual(BindType::Tmpfs, "/tmp");
    b.add_virtual(BindType::Tmpfs, "/run");
}

fn bind_workspace_and_home(b: &mut Bubblewrap, cave: &Cave, host_home: &Path) -> Result<()> {
    b.add_bind(BindType::Bind, &cave.workspace);
    if !cave.homedir.exists() {
        std::fs::create_dir_all(&cave.homedir).context("Failed to create cave home directory")?;
    }
    b.add_map_bind(BindType::Bind, &cave.homedir, host_home);
    Ok(())
}

fn bind_pilocal_and_caches(
    b: &mut Bubblewrap, 
    config: &Config, 
    cave: &Cave, 
    variant: Option<&str>, 
    writable: bool,
    internal_pilocal: &Path
) -> Result<()> {
    let host_pilocal = config.pilocal_path(&cave.name, variant);
    if !host_pilocal.exists() {
        std::fs::create_dir_all(&host_pilocal).context("Failed to create .pilocal directory")?;
    }
    let bind_type = if writable { BindType::Bind } else { BindType::RoBind };
    b.add_map_bind(bind_type, &host_pilocal, internal_pilocal);

    if config.cache_dir.exists() {
        b.add_bind(bind_type, &config.cache_dir);
    }
    if config.config_dir.exists() {
        b.add_bind(BindType::RoBind, &config.config_dir);
    }
    Ok(())
}

fn setup_xdg_runtime(b: &mut Bubblewrap) {
    if let Ok(runtime_dir) = env::var("XDG_RUNTIME_DIR") {
        b.add_bind(BindType::Bind, &runtime_dir);
        b.set_env("XDG_RUNTIME_DIR", &runtime_dir);
    }
}

fn setup_environment(b: &mut Bubblewrap, config: &Config, cave: &Cave, host_home: &Path, internal_pilocal: &Path) {
    b.set_env("HOME", host_home.to_str().unwrap());
    b.set_env("USER", &config.get_user());
    b.set_env("PI_WORKSPACE", cave.workspace.to_str().unwrap());
    b.set_env("PI_CAVE", &cave.name);
    
    let pilocal_bin = internal_pilocal.join("bin");
    b.add_env_first("PATH", "/usr/bin:/bin");
    b.add_env_first("PATH", host_home.join(".local").join("bin").to_str().unwrap());
    b.add_env_first("PATH", host_home.join(".cargo").join("bin").to_str().unwrap());
    b.add_env_first("PATH", host_home.join(".mix").join("escripts").to_str().unwrap());
    b.add_env_first("PATH", pilocal_bin.to_str().unwrap());
}

fn apply_custom_envs(
    b: &mut Bubblewrap, 
    pkg_envs: HashMap<String, String>, 
    cave_envs: &HashMap<String, String>,
    host_home: &Path,
    internal_pilocal: &Path
) {
    let resolve = |v: String| {
        v.replace("$/", &format!("{}/", internal_pilocal.display()))
         .replace("$", internal_pilocal.to_str().unwrap())
         .replace("@HOME", host_home.to_str().unwrap())
    };

    for (k, v) in pkg_envs {
        b.set_env(&k, &resolve(v));
    }
    for (k, v) in cave_envs {
        b.set_env(k, &resolve(v.clone()));
    }
}

fn execute_run(config: &Config, variant_opt: Option<String>, command: Vec<String>) -> Result<()> {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (_path, cave) = Cave::find_in_ancestry(&current_dir).context("no cave found")?;

    let (variant, final_command) = match variant_opt {
        Some(v) if v.starts_with(':') => (Some(v), command),
        Some(v) => {
            let mut new_cmd = vec![v];
            new_cmd.extend(command);
            (None, new_cmd)
        }
        None => (None, command),
    };

    let package_envs = crate::commands::cave::build::execute_build(config, &cave, variant.as_deref())?;

    let mut b = prepare_sandbox(SandboxOptions {
        config,
        cave: &cave,
        variant: variant.as_deref(),
        package_envs,
        writable_pilocal: false,
        dependency_dirs: Vec::new(),
    })?;
    
    log::info!("entering cave");
    if log::log_enabled!(log::Level::Info) {
        crate::commands::cave::info::run(config);
    }

    if !final_command.is_empty() {
        b.set_command(&final_command[0], &final_command[1..]);
    } else {
        b.set_command("/bin/bash", &[]);
    }

    b.exec()
}
