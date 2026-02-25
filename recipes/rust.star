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
    ok, version, matched_target = extract(pattern, v_tmp)
    
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

    subfolder = component_map.get(package_name)
    if subfolder == None:
        if package_name == "rust-std":
            subfolder = "rust-std-" + actual_target
        else:
            subfolder = package_name

    component_root = top_dir
    if subfolder:
        component_root = top_dir + "/" + subfolder

    if package_name == "rust-src":
        src_base = component_root + "/lib/rustlib/src/rust"
        v.export_link(src_base + "/*", "lib/rustlib/src/rust")
        v.export_env("RUST_SRC_PATH", "$/lib/rustlib/src/rust/library")
    elif package_name == "rust-std":
        std_base = component_root + "/lib/rustlib/" + target + "/lib"
        v.export_link(std_base + "/*", "lib/rustlib/" + target + "/lib")
    elif package_name == "rust":
        v.export_link(component_root + "/bin/*", "bin")
        v.export_link(component_root + "/lib/*", "lib")
        v.export_env("RUSTC_SYSROOT", "$")
        v.export_env("RUSTFLAGS", "--sysroot=$")
    else:
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
        
        v = create_version(package_name, version, release_date = date, release_type = channel)
        v.fetch(dl_url, checksum = target_data.get("hash"), filename = filename)
        v.extract()
        add_rust_component(v, package_name, target, top_dir)
        
        v.register()

def cargo_discovery(manager, package):
    url = "https://crates.io/api/v1/crates/" + package
    content = download(url)
    if not content:
        return
    doc = parse_json(content)
    data = doc.root

    versions = data.get("versions")
    if not versions:
        return
    for i in range(len(versions)):
        v_data = versions[i]
        if v_data.get("yanked").text() == "true":
            continue
        version = v_data["num"]
        
        v = create_version(package, version, release_date = v_data["created_at"])
        v.run("cargo install --root ~/.pilocal " + package + " --version " + version)
        # Note: cargo install handles exports by putting them in ~/.pilocal/bin
        # which is already in PATH in our cave setup.
        
        v.register()

COMPONENTS = ["rust", "cargo", "rust-analyzer", "rust-src", "rustfmt", "clippy", "rustc", "rust-std"]
for c in COMPONENTS:
    add_package(c, discover_rust_component)

add_manager("cargo", cargo_discovery)
