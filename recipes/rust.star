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

def parse_rust_filename(manifest_name, target, filename):
    # Remove extensions like .tar.gz, .zip etc.
    ok_base, v_tmp = extract(r"(.*)\.(?:tar\.gz|tar\.xz|zip|tar\.bz2)", filename)
    if not ok_base:
        v_tmp = filename

    top_dir = v_tmp

    # Pattern: manifest_name(-preview)?-(version)(-(target))?
    # We use escaping for manifest_name if it contains special chars, though rust-src is fine.
    pattern = manifest_name + "(?:-preview)?-([0-9.]+)(?:-(.*))?"
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
    for prefix in [manifest_name + "-preview", manifest_name]:
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
        elif package_name == "llvm-tools":
            # llvm-tools puts its binaries in lib/rustlib/<target>/bin/
            llvm_bin_rel = "lib/rustlib/" + target + "/bin"
            llvm_bin_abs = component_root + "/" + llvm_bin_rel
            v.export_link(llvm_bin_abs + "/*", "bin")
            
            # Set environment variables for cargo-llvm-cov and others
            # $/ resolves to the root of the .pilocal mount in the sandbox
            v.export_env("LLVM_COV", "$/bin/llvm-cov")
            v.export_env("LLVM_PROFDATA", "$/bin/llvm-profdata")
        else:
            # For most components (rust-analyzer, clippy, etc), we just need the bin folder
            v.export_link(component_root + "/bin/*", "bin")

def discover_rust_component(package_name, manifest_name):
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

        pkg = pkgs.get(manifest_name) or pkgs.get(manifest_name + "-preview")
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

        top_dir, version = parse_rust_filename(manifest_name, target, filename)

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

def discover_rust_all(_package_name):
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

        # We use 'rust' version as the base version for 'rust-all'
        rust_pkg = pkgs.get("rust")
        if not rust_pkg:
            continue

        target_data = rust_pkg.get("target").get(target) or rust_pkg.get("target").get("*")
        if not target_data or not target_data.get("available"):
            continue

        ok_file, filename = extract(r".*/([^/]+)$", target_data.get("url"))
        _, version = parse_rust_filename("rust", target, filename)

        v = create_version("rust-all")
        v.inspect(version)
        v.set_release_date(date)
        if channel != "stable":
            v.set_release_type("testing" if channel == "beta" else "unstable")

        for (c, _) in COMPONENTS:
            v.require_version(c, version)

        v.register()

def discover_rust(p): return discover_rust_component("rust", "rust")
def discover_cargo(p): return discover_rust_component("cargo", "cargo")
def discover_rust_analyzer(p): return discover_rust_component("rust-analyzer", "rust-analyzer")
def discover_rust_src(p): return discover_rust_component("rust-src", "rust-src")
def discover_rustfmt(p): return discover_rust_component("rustfmt", "rustfmt")
def discover_clippy(p): return discover_rust_component("clippy", "clippy")
def discover_rustc(p): return discover_rust_component("rustc", "rustc")
def discover_rust_std(p): return discover_rust_component("rust-std", "rust-std")
def discover_llvm_tools(p): return discover_rust_component("llvm-tools", "llvm-tools")

COMPONENTS = [
    ("rust", discover_rust),
    ("cargo", discover_cargo),
    ("rust-analyzer", discover_rust_analyzer),
    ("rust-src", discover_rust_src),
    ("rustfmt", discover_rustfmt),
    ("clippy", discover_clippy),
    ("rustc", discover_rustc),
    ("rust-std", discover_rust_std),
    ("llvm-tools", discover_llvm_tools)
]

for (pkg, func) in COMPONENTS:
    add_package(pkg, func)

add_package("rust-all", discover_rust_all)
add_manager("cargo", cargo_discovery)
