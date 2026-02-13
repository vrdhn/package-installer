use crate::models::config::Config;
use crate::models::cave::Cave;
use crate::services::sandbox::{Bubblewrap, BindType};
use std::env;
use std::path::Path;
use anyhow::{Context, Result};

pub fn run(config: &Config, variant: Option<String>, command: Vec<String>) {
    if let Err(e) = execute_run(config, variant, command) {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    }
}

fn execute_run(config: &Config, variant_opt: Option<String>, mut command: Vec<String>) -> Result<()> {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let (_path, cave) = Cave::find_in_ancestry(&current_dir)
        .context("No cave found in current directory or its ancestors.")?;

    // Detect if variant_opt is actually a variant or just the first part of the command
    let (variant, final_command) = if let Some(ref v) = variant_opt {
        if v.starts_with(':') {
            (Some(v.as_str()), command)
        } else {
            // It's not a variant, it's the command
            let mut new_cmd = vec![v.clone()];
            new_cmd.extend(command);
            (None, new_cmd)
        }
    } else {
        (None, command)
    };

    // Perform build before run and collect package-provided env vars
    let package_envs = crate::commands::cave::build::execute_build(config, &cave, variant)?;

    let settings = cave.get_effective_settings(variant)
        .context("Failed to get effective cave settings")?;

    let mut b = Bubblewrap::new();
    let host_home = config.get_host_home();

    // Basic flags
    b.add_flag("--unshare-pid");
    b.add_flag("--die-with-parent");

    // Standard Linux file system (ReadOnly)
    b.add_bind(BindType::RoBind, "/usr");
    b.add_bind(BindType::RoBind, "/lib");
    if Path::new("/lib64").exists() {
        b.add_bind(BindType::RoBind, "/lib64");
    }
    b.add_bind(BindType::RoBind, "/bin");
    b.add_bind(BindType::RoBind, "/sbin");
    b.add_bind(BindType::RoBind, "/etc");
    b.add_bind(BindType::RoBind, "/sys");

    // Virtual FS
    b.add_virtual(BindType::Proc, "/proc");
    b.add_virtual(BindType::Dev, "/dev");
    b.add_virtual(BindType::Tmpfs, "/tmp");
    b.add_virtual(BindType::Tmpfs, "/run");

    // Workspace (RW bind to same path)
    b.add_bind(BindType::Bind, &cave.workspace);

    // Cave Homedir (Mapped to host $HOME)
    if !cave.homedir.exists() {
        std::fs::create_dir_all(&cave.homedir).context("Failed to create cave home directory")?;
    }
    b.add_map_bind(BindType::Bind, &cave.homedir, &host_home);

    // .pilocal maintenance: cache -> RO mount in cave home
    let host_pilocal = config.pilocal_path(&cave.name, variant);
    if host_pilocal.exists() {
        b.add_map_bind(BindType::RoBind, &host_pilocal, host_home.join(".pilocal"));
    }

    // ~/.cache/pi and ~/.config/pi (ReadOnly bind)
    let host_cache_pi = config.cache_dir.clone();
    let host_config_pi = config.config_dir.clone();
    
    // In the sandbox, these are at the same path as host
    if host_cache_pi.exists() {
        b.add_bind(BindType::RoBind, &host_cache_pi);
    }
    if host_config_pi.exists() {
        b.add_bind(BindType::RoBind, &host_config_pi);
    }

    // XDG_RUNTIME_DIR handling
    if let Ok(runtime_dir) = env::var("XDG_RUNTIME_DIR") {
        b.add_bind(BindType::Bind, &runtime_dir);
        b.set_env("XDG_RUNTIME_DIR", &runtime_dir);
    }

    // Environment setup
    b.set_env("HOME", host_home.to_str().unwrap());
    b.set_env("USER", &config.get_user());
    b.set_env("PI_WORKSPACE", cave.workspace.to_str().unwrap());
    b.set_env("PI_CAVE", &cave.name);
    
    // PATH setup: .pilocal/bin from home should be first
    let pilocal_bin = host_home.join(".pilocal/bin");
    b.add_env_first("PATH", "/usr/bin:/bin");
    b.add_env_first("PATH", pilocal_bin.to_str().unwrap());

    // Apply package-provided environment variables
    for (k, v) in package_envs {
        // Resolve placeholders like @HOME
        let v = v.replace("@HOME", host_home.to_str().unwrap());
        b.set_env(&k, &v);
    }

    // Apply cave settings environment variables (overrides package-provided ones)
    for (k, v) in settings.set {
        let v = v.replace("@HOME", host_home.to_str().unwrap());
        b.set_env(&k, &v);
    }

    println!("you are now in cave");
    crate::commands::cave::info::run(config);

    if !final_command.is_empty() {
        b.set_command(&final_command[0], &final_command[1..]);
    } else {
        b.set_command("/bin/bash", &[]);
    }

    b.exec()
}
