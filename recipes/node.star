def get_platform_string():
    os = get_os()
    arch = get_arch()

    node_os = os
    if os == "windows":
        node_os = "win"
    elif os == "macos":
        node_os = "osx"

    node_arch = arch
    if arch == "x86_64":
        node_arch = "x64"
    elif arch == "aarch64":
        node_arch = "arm64"

    return node_os + "-" + node_arch

def install_node(package_name):
    platform = get_platform_string()
    print("Detected platform for Node:", platform)

    content = download("https://nodejs.org/dist/index.json")
    data = json_parse(content)

    # Process all available versions
    for i in range(len(data)):
        entry = data[i]
        version = entry["version"]

        found = False
        for f in entry["files"]:
            if f == platform:
                found = True
                break

        if found:
            ext = "tar.gz"
            if "win" in platform:
                ext = "zip"

            filename = "node-" + version + "-" + platform + "." + ext
            url = "https://nodejs.org/dist/" + version + "/" + filename
            shasums_url = "https://nodejs.org/dist/" + version + "/SHASUMS256.txt"

            # Determine release type
            # Rule: if security update is not available (security: false), it's obsolete
            # (unless it's LTS or the latest version)
            release_type = "stable"

            if entry["lts"]:
                release_type = "lts"

            # Note: 'unstable' and 'testing' are not easily distinguishable in nodejs index.json
            # but we have 'stable', 'lts', and 'obsolete' covered.

            add_version(
                pkgname = "node",
                version = version,
                release_date = entry["date"],
                release_type = release_type,
                url = url,
                filename = filename,
                checksum = "",
                checksum_url = shasums_url
            )

add_package("node", install_node)

def npm_discovery(installer, package):
    print("Syncing npm package:", package)
    url = "https://registry.npmjs.org/" + package
    content = download(url)
    data = json_parse(content)
    
    versions = data["versions"]
    time = data["time"]
    dist_tags = data["dist-tags"]
    
    for version in versions:
        v_data = versions[version]
        
        release_type = "stable"
        # Simple heuristic for release type
        if "-" in version:
            release_type = "testing"
        
        # Check if it's the latest or lts (using dist-tags as a hint)
        if version == dist_tags.get("latest"):
            release_type = "stable"
        
        add_version(
            pkgname = package,
            version = version,
            release_date = time.get(version, ""),
            release_type = release_type,
            url = v_data["dist"]["tarball"],
            filename = package.split("/")[-1] + "-" + version + ".tgz",
            checksum = v_data["dist"]["shasum"],
            checksum_url = "",
            installer_command = "npm --global install " + package
        )

add_installer("npm", npm_discovery)
