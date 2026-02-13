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

def parse_rust_filename(package_name, target, filename):
    """Parses version and top-level directory from a Rust component filename."""
    # Remove extensions
    v_tmp = filename
    for ext in [".tar.gz", ".tar.xz", ".zip", ".tar.bz2"]:
        if v_tmp.endswith(ext):
            v_tmp = v_tmp[:-len(ext)]
            break

    top_dir = v_tmp

    # Special case for rust-src: top_dir inside archive often lacks target triple
    if package_name == "rust-src":
        if target != "*" and top_dir.endswith("-" + target):
            top_dir = top_dir[:-(len(target) + 1)]

    # Extract version by stripping package prefix and target suffix
    v_parse = v_tmp
    if target != "*" and v_parse.endswith("-" + target):
        v_parse = v_parse[:-(len(target) + 1)]

    version = v_parse
    for prefix in [package_name + "-preview", package_name]:
        if v_parse.startswith(prefix + "-"):
            version = v_parse[len(prefix) + 1:]
            break

    return top_dir, version

def get_component_layout(package_name, target, top_dir):
    """Determines the internal subfolder, file mapping, and environment variables."""
    actual_target = target if target != "*" else ""
    
    # Map package names to their internal directory names if they differ from package_name
    component_map = {
        "rust": "rustc",
    }

    subfolder = component_map.get(package_name)
    if subfolder == None:
        if package_name == "rust-std":
            subfolder = "rust-std-" + actual_target
        else:
            # Most components (cargo, rust-analyzer, etc) just use the package name
            # as the subfolder name, even if the manifest uses -preview suffix.
            subfolder = package_name

    component_root = top_dir
    if subfolder:
        component_root = top_dir + "/" + subfolder

    # Default mapping and environment settings
    filemap = {component_root + "/bin/*": "bin"}
    env_vars = {
        "RUSTC_SYSROOT": "$",
        "RUSTFLAGS": "--sysroot=$"
    }

    if package_name == "rust-src":
        # rust-src contents are directly in top_dir/rust-src/lib/rustlib/src/rust
        src_base = component_root + "/lib/rustlib/src/rust"
        filemap = {src_base + "/*": "lib/rustlib/src/rust"}
        env_vars["RUST_SRC_PATH"] = "$/lib/rustlib/src/rust/library"
    elif package_name == "rust-std":
        # rust-std contents are in top_dir/rust-std-<target>/lib/rustlib/<target>/lib
        std_base = component_root + "/lib/rustlib/" + target + "/lib"
        filemap = {std_base + "/*": "lib/rustlib/" + target + "/lib"}
    elif package_name == "rust":
        # Main rust package (rustc) needs both bin and lib
        filemap = {
            component_root + "/bin/*": "bin",
            component_root + "/lib/*": "lib",
        }

    return filemap, env_vars

def discover_rust_component(package_name):
    target = get_rust_target()

    for channel in ["stable", "beta", "nightly"]:
        url = "https://static.rust-lang.org/dist/channel-rust-" + channel + ".toml"
        content = download(url)
        if not content:
            continue

        data = toml_parse(content)
        date = data.get("date", "")
        pkgs = data.get("pkg", {})

        # Look for the package (try preview if exact name not found)
        pkg = pkgs.get(package_name) or pkgs.get(package_name + "-preview")
        if not pkg:
            continue

        # Find target data (try wildcard if specific target not found)
        target_dict = pkg.get("target", {})
        target_data = target_dict.get(target) or target_dict.get("*")

        if not target_data or not target_data.get("available"):
            continue

        dl_url = target_data.get("url")
        if not dl_url:
            continue

        filename = dl_url.split('/')[-1]
        top_dir, version = parse_rust_filename(package_name, target, filename)
        filemap, env_vars = get_component_layout(package_name, target, top_dir)

        add_version(
            pkgname = package_name,
            version = version,
            release_date = date,
            release_type = channel,
            url = dl_url,
            filename = filename,
            checksum = target_data.get("hash", ""),
            checksum_url = "",
            filemap = filemap,
            env = env_vars
        )

def cargo_discovery(manager, package):
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
            filemap = {"bin/*": "bin"}
        )

# Register toolchain components
COMPONENTS = ["rust", "cargo", "rust-analyzer", "rust-src", "rustfmt", "clippy", "rustc", "rust-std"]
for c in COMPONENTS:
    add_package(c, discover_rust_component)

# Add manager for cargo packages
add_manager("cargo", cargo_discovery)
