use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;

pub fn run(config: &Config) {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    if let Some((path, cave)) = Cave::find_in_ancestry(&current_dir) {
        let active_status = if config.is_inside_cave() { " (ACTIVE)" } else { "" };
        println!("Cave Name: {}{}", cave.name, active_status);
        println!("Cave File: {}", path.display());
        println!("Workspace: {}", cave.workspace.display());
        println!("Home:      {}", cave.homedir.display());
        
        println!("
Default Settings:");
        println!("  Packages: {:?}", cave.settings.packages);
        println!("  Set:      {:?}", cave.settings.set);
        println!("  Unset:    {:?}", cave.settings.unset);

        if !cave.variants.is_empty() {
            println!("
Variants:");
            for (name, settings) in &cave.variants {
                println!("  :{}", name);
                println!("    Packages: {:?}", settings.packages);
                println!("    Set:      {:?}", settings.set);
                println!("    Unset:    {:?}", settings.unset);
            }
        }
    } else {
        println!("No cave found in current directory or its ancestors.");
    }
}
