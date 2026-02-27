use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;

pub fn run(config: &Config) {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    if let Some((path, cave)) = Cave::find_in_ancestry(&current_dir) {
        let active_status = if config.is_inside_cave() { " (ACTIVE)" } else { "" };
        println!("name: {}{}", cave.name, active_status);
        println!("file: {}", path.display());
        println!("work: {}", cave.workspace.display());
        println!("home: {}", config.state_home_dir.display());
        
        println!("\nsettings:");
        println!("  pkgs: {:?}", cave.settings.packages);
        println!("  set:  {:?}", cave.settings.set);
        println!("  uns:  {:?}", cave.settings.unset);

        if !cave.variants.is_empty() {
            println!("\nvariants:");
            for (name, settings) in &cave.variants {
                println!("  :{}", name);
                println!("    pkgs: {:?}", settings.packages);
                println!("    set:  {:?}", settings.set);
                println!("    uns:  {:?}", settings.unset);
            }
        }
    } else {
        log::error!("no cave found");
    }
}
