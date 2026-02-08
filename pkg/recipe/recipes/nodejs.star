def map_file(file):
    mapping = {
        "linux-x64": ("linux", "x64"),
        "linux-arm64": ("linux", "arm64"),
        "osx-x64-tar": ("darwin", "x64"),
        "osx-arm64-tar": ("darwin", "arm64"),
        "win-x64-zip": ("windows", "x64"),
        "win-arm64-zip": ("windows", "arm64"),
    }
    return mapping.get(file)

def resolve_nodejs(pkg_name):
    print("resolve_nodejs called for", pkg_name)
    data = download(url = "https://nodejs.org/dist/index.json")
    print("downloaded data length:", len(data))
    versions = json.decode(data=data)
    print("decoded", len(versions), "versions")

    for v in versions:
        version = v["version"].lstrip("v")

        status = "current"
        if v.get("lts"):
            status = "lts"

        for file in v["files"]:
            info = map_file(file)
            if not info:
                continue

            os_type, arch_type = info

            # We assume nodejs.org has fixed extensions for these
            ext = ".tar.gz"
            if os_type == "windows":
                ext = ".zip"

            filename = "node-v{}-{}{}".format(version, file, ext)
            url = "https://nodejs.org/dist/v{}/{}".format(version, filename)

            add_version(
                name = "nodejs",
                version = version,
                release_status = status,
                release_date = v.get("date", ""),
                os = os_type,
                arch = arch_type,
                url = url,
                filename = filename,
                checksum = "",
                env = {},
                symlinks = {
                    "node-v{}-{}/bin/*".format(version, file): ".local/bin"
                }
            )

add_pkgdef(regex="nodejs", handler=resolve_nodejs)
