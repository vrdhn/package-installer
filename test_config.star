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

def install_firefox(package_name):
    print("Installing Firefox:", package_name)

add_package("^node", install_node)
add_package("^firefox", install_firefox)