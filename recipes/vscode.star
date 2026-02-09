def resolve_vscode(pkg_name):
    # For VS Code, we primarily fetch the latest version for each platform
    # The update API provides the specific download URL and SHA256

    platforms = [
        {"os": "linux", "arch": "x64", "code_platform": "linux-x64", "bin_path": "VSCode-linux-x64/bin/code"},
        {"os": "linux", "arch": "arm64", "code_platform": "linux-arm64", "bin_path": "VSCode-linux-arm64/bin/code"},
        {"os": "darwin", "arch": "x64", "code_platform": "darwin", "bin_path": "Visual Studio Code.app/Contents/Resources/app/bin/code"},
        {"os": "darwin", "arch": "arm64", "code_platform": "darwin-arm64", "bin_path": "Visual Studio Code.app/Contents/Resources/app/bin/code"},
    ]

    for p in platforms:
        # Using 'latest' gives us the most recent stable release info
        update_url = "https://update.code.visualstudio.com/api/update/%s/stable/latest" % p["code_platform"]

        res_data = download(url = update_url)
        if not res_data:
            continue

        info = json.decode(data = res_data)
        if not info or "url" not in info:
            continue

        # productVersion is the human-readable version (e.g., "1.109.0")
        version = info.get("productVersion", "latest")
        url = info["url"]
        filename = url.split("/")[-1]

        checksum = ""
        if "sha256hash" in info:
            checksum = "sha256:" + info["sha256hash"]

        add_version(
            name = "vscode",
            version = version,
            release_status = "stable",
            release_date = "",
            os = p["os"],
            arch = p["arch"],
            url = url,
            filename = filename,
            checksum = checksum,
            env = {},
            symlinks = {
                p["bin_path"]: ".local/bin/code",
                p["bin_path"] + "-tunnel": ".local/bin/code-tunnel"
            }
        )

add_pkgdef(regex="vscode", handler=resolve_vscode)
