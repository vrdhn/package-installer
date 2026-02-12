use crate::models::config::Config;
use crate::models::cave::Cave;
use std::env;

pub fn run(_config: &Config) {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let cave_file = current_dir.join(Cave::FILENAME);

    if cave_file.exists() {
        println!("Cave already initialized in {}", current_dir.display());
        return;
    }

    let cave = Cave::new(current_dir.clone());
    cave.save(&cave_file).expect("Failed to save cave file");
    println!("Initialized new cave in {}", current_dir.display());
}
