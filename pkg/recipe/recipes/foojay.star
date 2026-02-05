def resolve_foojay(pkg_name):
    # Fetch all GA JDKs
    data = download(url = "https://api.foojay.io/disco/v3.0/packages?package_type=jdk&release_status=ga")
    resp = json.decode(data)

    for p in resp["result"]:
        version = p["java_version"]

        status = p.get("release_status", "unknown")
        if status == "ga":
            status = "stable"

        filename = p["filename"]

        # Map OS names if necessary
        os_raw = p["operating_system"]
        os_type = os_raw
        if os_raw == "macos":
            os_type = "darwin"

        # Map Arch names if necessary
        arch_raw = p["architecture"]
        arch_type = arch_raw
        if arch_raw == "aarch64":
            arch_type = "arm64"
        elif arch_raw == "amd64": # Just in case
            arch_type = "x64"

        add_version(
            name = "java-" + p["distribution"],
            version = version,
            release_status = status,
            release_date = p.get("release_date", ""),
            os = os_type,
            arch = arch_type,
            url = p["links"]["pkg_download_redirect"],
            filename = filename,
            checksum = p.get("checksum", ""),
            env = {},
            symlinks = {}
        )

add_pkgdef("foojay:.*", resolve_foojay)
