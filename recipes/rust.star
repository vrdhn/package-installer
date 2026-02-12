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

        pitree_root = ""
        if package_name == "rust":
                pitree_root = package_name + "-" + version + "-" + target + "/" + "rustc"
        else:
                pitree_root = package_name + "-" + version + "-" + target + "/" + package_name

        add_version(
            pkgname = package_name,
            version = version,
            release_date = date,
            release_type = channel,
            url = dl_url,
            filename = filename,
            checksum = checksum,
            checksum_url = "",
            filemap = {pitree_root +  "/bin/*": "bin"}
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
