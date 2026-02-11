use std::process::Command;

fn main() {
    let version = Command::new("git")
        .args(&["describe", "--always", "--dirty", "--tags"])
        .output()
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .unwrap_or_else(|_| "unknown".to_string());

    let date = Command::new("date")
        .output()
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .unwrap_or_else(|_| "unknown".to_string());

    println!("cargo:rustc-env=BUILD_VERSION={}", version);
    println!("cargo:rustc-env=BUILD_DATE={}", date);
    println!("cargo:rerun-if-changed=build.rs");
    println!("cargo:rerun-if-changed=.git/HEAD");
}
