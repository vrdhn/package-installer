def get_rust_target():
    os = get_os()
    arch = get_arch()
    tos = {"linux": "unknown-linux-gnu", "macos": "apple-darwin", "windows": "pc-windows-msvc"}.get(os, "unknown-linux-gnu")
    return arch + "-" + tos

def parse_rust_version(base_name, target, filename):
    ok, base = extract(r"(.*)\.(?:tar\.gz|tar\.xz|zip|tar\.bz2)", filename)
    if not ok: base = filename

    pattern = base_name + "(?:-preview)?-([0-9.]+)(?:-(.*))?"
    ok, version, _ = extract(pattern, base)
    if ok: return base, version

    v = base
    if target != "*" and v.endswith("-" + target):
        v = v[:-(len(target) + 1)]
    
    for prefix in [base_name + "-preview", base_name]:
        if v.startswith(prefix + "-"):
            return base, v[len(prefix) + 1:]
    
    return base, v

# Component configuration: (dir_patterns, links, env_vars)
RUSTC_COMPONENTS = {
    "rustc": (["rustc"], [("bin/*", "bin"), ("lib/*", "lib")], [("RUSTC_SYSROOT", "$")]),
    "cargo": (["cargo", "cargo-preview"], [("bin/*", "bin")], []),
    "rustfmt": (["rustfmt", "rustfmt-preview"], [("bin/*", "bin")], []),
    "clippy": (["clippy", "clippy-preview"], [("bin/*", "bin")], []),
    "rust-analyzer": (["rust-analyzer", "rust-analyzer-preview"], [("bin/*", "bin")], []),
    "rust-std": (["rust-std-{target}"], [("lib/rustlib/{target}/lib/*", "lib/rustlib/{target}/lib")], []),
    "llvm-tools": (
        ["llvm-tools", "llvm-tools-preview"],
        [("lib/rustlib/{target}/bin/*", "bin")],
        [("LLVM_COV", "$/bin/llvm-cov"), ("LLVM_PROFDATA", "$/bin/llvm-profdata")]
    ),
    "llvm-bitcode-linker": (
        ["llvm-bitcode-linker", "llvm-bitcode-linker-preview"],
        [("lib/rustlib/{target}/bin/self-contained/llvm-bitcode-linker", "bin/llvm-bitcode-linker")],
        []
    ),
    "rust-docs": (["rust-docs", "rust-docs-preview"], [("share/doc/rust/html/*", "share/doc/rust/html")], []),
    "rust-docs-json": (["rust-docs-json", "rust-docs-json-preview"], [("share/doc/rust/json/*", "share/doc/rust/json")], []),
    "rust-analysis": (["rust-analysis-{target}"], [("lib/rustlib/{target}/analysis/*", "lib/rustlib/{target}/analysis")], []),
}

RUST_SRC_CONFIG = (["rust-src"], [("lib/rustlib/src/rust/*", "lib/rustlib/src/rust")], [("RUST_SRC_PATH", "$/lib/rustlib/src/rust/library")])

def _apply_config(v, target, top_dir, dir_patterns, links, envs):
    for pat in dir_patterns:
        root = top_dir + "/" + pat.format(target=target)
        for src, dest in links:
            v.export_link(root + "/" + src.format(target=target), dest.format(target=target))
        for env_name, env_val in envs:
            v.export_env(env_name, env_val)

def _fetch_and_prepare(package_name, channel, target):
    content = download("https://static.rust-lang.org/dist/channel-rust-" + channel + ".toml")
    if not content: return None
    data = parse_toml(content).root
    pkg = data.get("pkg").get(package_name)
    if not pkg: return None
    target_data = pkg.get("target").get(target) or pkg.get("target").get("*")
    if not target_data or not target_data.get("available"): return None

    url = target_data.get("url")
    _ok, filename = extract(r".*/([^/]+)$", url)
    top_dir, version = parse_rust_version(package_name, target, filename)

    v = create_version(package_name)
    v.inspect(version)
    v.set_release_date(data.attribute("date") or "")
    if channel != "stable":
        v.set_release_type("testing" if channel == "beta" else "unstable")

    v.fetch(url, checksum = target_data.get("hash"), filename = filename)
    v.extract()
    return v, top_dir, filename

def discover_rust(p):
    target = get_rust_target()
    for channel in ["stable", "beta", "nightly"]:
        res = _fetch_and_prepare("rust", channel, target)
        if not res: continue
        v, top_dir, _ = res

        for name in RUSTC_COMPONENTS:
            dir_patterns, links, envs = RUSTC_COMPONENTS[name]
            _apply_config(v, target, top_dir, dir_patterns, links, envs)
        v.register()

def discover_rust_src(p):
    for channel in ["stable", "beta", "nightly"]:
        res = _fetch_and_prepare("rust-src", channel, "*")
        if not res: continue
        v, top_dir, _ = res
        dir_patterns, links, envs = RUST_SRC_CONFIG
        _apply_config(v, "*", top_dir, dir_patterns, links, envs)
        v.register()

def cargo_discovery(_manager, package):
    content = download("https://crates.io/api/v1/crates/" + package)
    if not content: return
    root = parse_json(content).root
    crate = root.get("crate")
    if not crate: return
    latest = crate.get("max_version") or crate.get("newest_version")
    for v_data in (root.get("versions") or []):
        if v_data.get("yanked"): continue
        version = v_data.get("num")
        v = create_version("cargo:" + package)
        v.inspect(version)
        v.set_release_date(v_data.get("created_at") or "")
        v.require("rust")
        if version == latest: v.set_release_type("stable")
        v.run("cargo install " + package + " --version " + version + " --locked --root ~/.pilocal")
        v.register()

add_package("rust", discover_rust)
add_package("rust-src", discover_rust_src)
add_manager("cargo", cargo_discovery)
