def resolve_golang(pkg_name):
    data = download(url = "https://go.dev/dl/?mode=json&include=all")
    releases = json.decode(data)

    #  {
    #  "version": "go1.26rc2",
    #  "stable": false,
    #  "files": [
    #   {
    #    "filename": "go1.26rc2.src.tar.gz",
    #    "os": "",
    #    "arch": "",
    #    "version": "go1.26rc2",
    #    "sha256": "e25cc8c5ffe1241a5d87199209243d70c24847260fb1ea7b163a95b537de65ac",
    #    "size": 34091929,
    #    "kind": "source"
    #   },
    #

    for release in releases:
        # release["version"] is like "go1.21.6"
        version_str = release["version"]
        if version_str.startswith("go"):
            version_str = version_str[2:]

        status = "unstable"
        if release.get("stable", False):
            status = "stable"

        for file in release["files"]:
            if file.get("kind") != "archive":
                continue

            # Map OS
            go_os = file["os"]
            os_type = "unknown"
            if go_os == "linux":
                os_type = "linux"
            elif go_os == "darwin":
                os_type = "darwin"
            elif go_os == "windows":
                os_type = "windows"

            # Map Arch
            go_arch = file["arch"]
            arch_type = "unknown"
            if go_arch == "amd64":
                arch_type = "x64"
            elif go_arch == "arm64":
                arch_type = "arm64"

            filename = file["filename"]

            add_version(
                name = "golang",
                version = version_str,
                release_status = status,
                release_date = release.get("time", ""),
                os = os_type,
                arch = arch_type,
                url = "https://go.dev/dl/" + filename,
                checksum = "sha256:" + file["sha256"],
                filename = filename,
                env = {
                    "GOROOT": "${PI_PKG_ROOT}",
                    "GOPATH": "~/go"
                },
                symlinks = {
                    "go/bin/*": ".local/bin"
                }
            )

add_pkgdef("golang", resolve_golang)
