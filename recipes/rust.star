def resolve_rust(pkg_name):
    # Rust releases are on GitHub but binaries are on static.rust-lang.org
    # For now, we'll use GitHub to find versions
    releases = download_github_releases(owner = "rust-lang", repo = "rust")

    for release in releases:
        version = release["tag_name"]
        if version.startswith("v"):
            version = version[1:]

        # Filter out RCs unless explicitly requested (future)
        status = "stable"
        if "rc" in version:
            status = "rc"

        # Rust standalone installers follow a specific pattern:
        # https://static.rust-lang.org/dist/rust-1.75.0-x86_64-unknown-linux-gnu.tar.gz

        platforms = [
            {"os": "linux", "arch": "x64", "triple": "x86_64-unknown-linux-gnu"},
            {"os": "linux", "arch": "arm64", "triple": "aarch64-unknown-linux-gnu"},
            {"os": "darwin", "arch": "x64", "triple": "x86_64-apple-darwin"},
            {"os": "darwin", "arch": "arm64", "triple": "aarch64-apple-darwin"},
        ]

        for p in platforms:
            filename = "rust-%s-%s.tar.gz" % (version, p["triple"])
            url = "https://static.rust-lang.org/dist/%s" % filename

            add_version(
                name = "rust",
                version = version,
                release_status = status,
                release_date = release["published_at"][:10],
                os = p["os"],
                arch = p["arch"],
                url = url,
                filename = filename,
                checksum = "", # We'd need to fetch .sha256 file
                env = {
                    "PATH": "${PI_PKG_ROOT}/rustc/bin:${PATH}"
                },
                symlinks = {
                    "rustc/bin/*": ".local/bin",
                    "cargo/bin/*": ".local/bin"
                }
            )

add_pkgdef(regex="rust", handler=resolve_rust)
