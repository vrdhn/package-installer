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
        
        is_stable = release.get("stable", False)
        
        match = False
        if version_query == "latest" or version_query == "":
            match = True # Take first one
        elif version_query == "stable":
            if is_stable:
                match = True
        elif version_str.startswith(version_query):
            match = True

        if not match:
            continue

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
            
            if os_type != context.os:
                continue

            # Map Arch
            go_arch = file["arch"]
            arch_type = "unknown"
            if go_arch == "amd64":
                arch_type = "x64"
            elif go_arch == "arm64":
                arch_type = "arm64"
            
            if arch_type != context.arch:
                continue

            filename = file["filename"]
            
            # Check extensions
            supported = False
            for allowed in context.extensions:
                if filename.endswith(allowed):
                    supported = True
                    break
            
            if not supported:
                continue

            pkgs.append({
                "name": "golang",
                "version": version_str,
                "os": os_type,
                "arch": arch_type,
                "url": "https://go.dev/dl/" + filename,
                "filename": filename,
                "env": {
                    "GOROOT": "${PI_PKG_ROOT}",
                    "GOPATH": "~/go"
                }
            })

    return pkgs
