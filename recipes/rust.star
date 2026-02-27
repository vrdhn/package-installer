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
    # Remove extensions like .tar.gz, .zip etc.
    ok_base, v_tmp = extract(r"(.*)\.(?:tar\.gz|tar\.xz|zip|tar\.bz2)", filename)
    if not ok_base:
        v_tmp = filename

    top_dir = v_tmp

    # Pattern: package_name(-preview)?-(version)(-(target))?
    # We use escaping for package_name if it contains special chars, though rust-src is fine.
    pattern = package_name + "(?:-preview)?-([0-9.]+)(?:-(.*))?"
    ok, version, _ = extract(pattern, v_tmp)

    if ok:
        # If it's rust-src, it might not have a target in the filename
        # If it has a target, it's at the end.
        return top_dir, version

    # Fallback to manual if regex fails for some reason
    v_parse = v_tmp
    if target != "*" and v_parse.endswith("-" + target):
        v_parse = v_parse[:-(len(target) + 1)]

    version = v_parse
    for prefix in [package_name + "-preview", package_name]:
        if v_parse.startswith(prefix + "-"):
            version = v_parse[len(prefix) + 1:]
            break

    return top_dir, version

def add_rust_component(v, package_name, target, top_dir):
    actual_target = target if target != "*" else ""

    component_map = {
        "rust": "rustc",
    }

    # Common subfolder patterns in Rust components
    subfolders = []
    if package_name in component_map:
        subfolders.append(component_map[package_name])

    if package_name == "rust-std":
        subfolders.append("rust-std-" + actual_target)
    else:
        # Try both the package name and package-target
        subfolders.append(package_name)
        if actual_target != "":
             subfolders.append(package_name + "-" + actual_target)

    if package_name == "rust-src":
        # rust-src is special and doesn't use the 'bin' export
        component_root = top_dir + "/" + package_name
        src_base = component_root + "/lib/rustlib/src/rust"
        v.export_link(src_base + "/*", "lib/rustlib/src/rust")
        v.export_env("RUST_SRC_PATH", "$/lib/rustlib/src/rust/library")
        return

    for sub in subfolders:
        component_root = top_dir + "/" + sub

        if package_name == "rust-std":
            std_base = component_root + "/lib/rustlib/" + target + "/lib"
            v.export_link(std_base + "/*", "lib/rustlib/" + target + "/lib")
        elif package_name == "rust":
            v.export_link(component_root + "/bin/*", "bin")
            v.export_link(component_root + "/lib/*", "lib")
            v.export_env("RUSTC_SYSROOT", "$")
            v.export_env("RUSTFLAGS", "--sysroot=$")
        else:
            # For most components (rust-analyzer, clippy, etc), we just need the bin folder
            v.export_link(component_root + "/bin/*", "bin")

def discover_rust_component(package_name):
    target = get_rust_target()

    for channel in ["stable", "beta", "nightly"]:
        url = "https://static.rust-lang.org/dist/channel-rust-" + channel + ".toml"
        content = download(url)
        if not content:
            continue

        doc = parse_toml(content)
        data = doc.root
        date = data.attribute("date") or ""
        pkgs = data.get("pkg")

        pkg = pkgs.get(package_name) or pkgs.get(package_name + "-preview")
        if not pkg:
            continue

        target_dict = pkg.get("target")
        target_data = target_dict.get(target) or target_dict.get("*")

        if not target_data or not target_data.get("available"):
            continue

        dl_url = target_data.get("url")
        if not dl_url:
            continue

        ok_file, filename = extract(r".*/([^/]+)$", dl_url)
        if not ok_file:
            filename = dl_url.split('/')[-1]

        top_dir, version = parse_rust_filename(package_name, target, filename)

        v = create_version(package_name)
        v.inspect(version)
        v.set_release_date(date)
        if channel != "stable":
            v.set_release_type("testing" if channel == "beta" else "unstable")

        v.fetch(dl_url, checksum = target_data.get("hash"), filename = filename)
        v.extract()
        add_rust_component(v, package_name, target, top_dir)

        v.register()

def cargo_discovery(_manager, package):
    url = "https://crates.io/api/v1/crates/" + package
    content = download(url)
    if not content:
        return

    doc = parse_json(content)
    root = doc.root

    crate_node = root.get("crate")
    if not crate_node:
        return
    latest_version = crate_node.get("max_version")
    if not latest_version:
         latest_version = crate_node.get("newest_version")

    versions = root.get("versions")
    if not versions:
        return

    for v_data in versions:
        if v_data.get("yanked"):
            continue

        version = v_data.get("num")
        if not version:
            continue

        v = create_version("cargo:" + package)
        v.inspect(version)
        v.set_release_date(v_data.get("created_at") or "")

        v.require("rust")
        v.require("cargo")
        v.require("rust-std")

        if version == latest_version:
            v.set_release_type("stable")

        v.run("cargo install " + package + " --version " + version + " --locked --root ~/.pilocal")

        v.register()

COMPONENTS = ["rust", "cargo", "rust-analyzer", "rust-src", "rustfmt", "clippy", "rustc", "rust-std"]
for c in COMPONENTS:
    add_package(c, discover_rust_component)

add_manager("cargo", cargo_discovery)
