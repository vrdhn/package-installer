def discover(pkg_name, version_query, context):
    return {
        "url": "https://go.dev/dl/?mode=json&include=all",
        "method": "GET"
    }

def parse(pkg_name, data, version_query, context):
    releases = json.decode(data)
    pkgs = []

    for release in releases:
        # release["version"] is like "go1.21.6"
        version_str = release["version"]
        if version_str.startswith("go"):
            version_str = version_str[2:]
        
        status = "unstable"
        if release.get("stable", False):
            status = "stable"
        
        # We ignore version_query filtering here and return everything
        # The resolver will filter by version if needed.
        # Note: 'stable' query logic is lost if we don't filter here, 
        # but the instruction was "return every version".

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
            
            pkgs.append({
                "name": "golang",
                "version": version_str,
                "release_status": status,
                "os": os_type,
                "arch": arch_type,
                "url": "https://go.dev/dl/" + filename,
                "filename": filename,
                "env": {
                    "GOROOT": "${PI_PKG_ROOT}",
                    "GOPATH": "~/go"
                },
                "symlinks": {
                    "go/bin/*": ".local/bin"
                }
            })

    return pkgs
