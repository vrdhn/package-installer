def get_rust_target():
    os = get_os()
    arch = get_arch()
    
    triple_os = "unknown-linux-gnu"
    if os == "macos":
        triple_os = "apple-darwin"
    elif os == "windows":
        triple_os = "pc-windows-msvc"
    
    triple_arch = arch
    if arch == "x86_64":
        triple_arch = "x86_64"
    elif arch == "aarch64":
        triple_arch = "aarch64"
    
    return triple_arch + "-" + triple_os

def discover_rust_component(package_name):
    target = get_rust_target()
    # rust-src is target-independent
    if package_name == "rust-src":
        target = None
        
    for channel in ["stable", "beta", "nightly"]:
        url = "https://static.rust-lang.org/dist/channel-rust-" + channel + ".toml"
        print("Fetching manifest:", url)
        content = download(url)
        if not content:
            continue
        
        data = toml_parse(content)
        date = data.get("date", "")
        
        pkg = data.get("pkg", {}).get(package_name)
        if not pkg:
            continue
            
        version_full = pkg.get("version", "")
        if not version_full:
            continue
            
        version = version_full.split(" ")[0]
        
        # Check if target is supported for this component
        if target:
            target_data = pkg.get("target", {}).get(target)
            if not target_data or not target_data.get("available"):
                continue
            
            filename = package_name + "-" + version + "-" + target + ".tar.gz"
        else:
            filename = package_name + "-" + version + ".tar.gz"
                
        # Construct download URL
        dl_url = "https://static.rust-lang.org/dist/" + filename
        if channel != "stable" and date:
            dl_url = "https://static.rust-lang.org/dist/" + date + "/" + filename
            
        add_version(
            pkgname = package_name,
            version = version,
            release_date = date,
            release_type = channel,
            url = dl_url,
            filename = filename,
            checksum = "",
            checksum_url = dl_url + ".sha256"
        )

def cargo_discovery(manager, package):
    print("Syncing cargo package:", package)
    url = "https://crates.io/api/v1/crates/" + package
    content = download(url)
    if not content:
        return
    data = json_parse(content)
    
    versions = data.get("versions", [])
    for v in versions:
        if v.get("yanked"):
            continue
        version = v["num"]
        add_version(
            pkgname = package,
            version = version,
            release_date = v["created_at"],
            release_type = "stable" if "-" not in version else "testing",
            url = "https://crates.io/api/v1/crates/" + package + "/" + version + "/download",
            filename = package + "-" + version + ".crate",
            checksum = v["checksum"],
            checksum_url = "",
            manager_command = "cargo install " + package + " --version " + version
        )

# Register toolchain components
COMPONENTS = ["rust", "cargo", "rust-analyzer", "rust-src", "rustfmt", "clippy", "rustc", "rust-std"]
for c in COMPONENTS:
    add_package(c, discover_rust_component)

# Add manager for cargo packages
add_manager("cargo", cargo_discovery)
