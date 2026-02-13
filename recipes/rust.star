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

    for channel in ["stable", "beta", "nightly"]:
        url = "https://static.rust-lang.org/dist/channel-rust-" + channel + ".toml"
        content = download(url)
        if not content:
            continue

        data = toml_parse(content)
        date = data.get("date", "")

        pkgs = data.get("pkg", {})
        pkg = pkgs.get(package_name)

        # Try -preview suffix (e.g. rustfmt -> rustfmt-preview)
        if not pkg:
            pkg = pkgs.get(package_name + "-preview")

        if not pkg:
            continue

        version_full = pkg.get("version", "")
        if not version_full:
            continue

        # Check if target is supported for this component
        target_dict = pkg.get("target", {})
        target_data = target_dict.get(target)
        if not target_data or not target_data.get("available"):
            # Try wildcard target
            target_data = target_dict.get("*")

        if not target_data or not target_data.get("available"):
            continue

        dl_url = target_data.get("url")
        checksum = target_data.get("hash")

        if not dl_url:
            continue

        filename = dl_url.split('/')[-1]

        # Parse version from filename: <prefix>-<version>-<target>.<ext>
        v_tmp = filename
        for ext in [".tar.gz", ".tar.xz", ".zip", ".tar.bz2"]:
            if v_tmp.endswith(ext):
                v_tmp = v_tmp[:-len(ext)]
                break

        # Remove target suffix if present
        if target != "*" and v_tmp.endswith("-" + target):
            v_tmp = v_tmp[:-(len(target) + 1)]

        # The version is what's left after removing the package prefix
        # We try to match the package name or its preview variant
        version = v_tmp
        for prefix in [package_name + "-preview", package_name]:
            if v_tmp.startswith(prefix + "-"):
                version = v_tmp[len(prefix) + 1:]
                break

        # Special casing for component subfolders
        component_map = {
            "rust": "rustc",
            "rust-src": "rust-src/lib/rustlib/src/rust",
            "rust-std": "rust-std-" + target + "/lib/rustlib/" + target + "/lib",
        }
        subfolder = component_map.get(package_name, package_name)
        
        pitree_root = package_name + "-" + version + "-" + target + "/" + subfolder

        # File mapping and environment logic
        filemap = {pitree_root + "/bin/*": "bin"}
        env_vars = {}

        if package_name == "rust-src":
            # rust-src extracts to lib/rustlib/src/rust
            # We map its contents to the same relative path in .pilocal
            filemap = {pitree_root + "/*": "lib/rustlib/src/rust"}
            env_vars = {"RUST_SRC_PATH": "$/lib/rustlib/src/rust/library"}
        elif package_name == "rust-std":
            filemap = {pitree_root + "/*": "lib/rustlib/" + target + "/lib"}
        elif package_name == "rust":
            # Main rust package might need LD_LIBRARY_PATH if it has libs in a non-standard place
            # but usually it finds them relative to bin/rustc
            pass

        add_version(
            pkgname = package_name,
            version = version,
            release_date = date,
            release_type = channel,
            url = dl_url,
            filename = filename,
            checksum = checksum,
            checksum_url = "",
            filemap = filemap,
            env = env_vars
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
            filemap = {"bin/*": "bin"},
            manager_command = "cargo install " + package + " --version " + version
        )

# Register toolchain components
COMPONENTS = ["rust", "cargo", "rust-analyzer", "rust-src", "rustfmt", "clippy", "rustc", "rust-std"]
for c in COMPONENTS:
    add_package(c, discover_rust_component)

# Add manager for cargo packages
add_manager("cargo", cargo_discovery)
