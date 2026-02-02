def discover(pkg_name, version_query, context):
    return {
        "url": "https://nodejs.org/dist/index.json",
        "method": "GET"
    }

def parse(pkg_name, data, version_query, context):
    versions = json.decode(data)
    pkgs = []
    
    for v in versions:
        version = v["version"].lstrip("v")
        if version_query != "latest" and version_query != "" and not version.startswith(version_query):
            continue
            
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
            
            # Final check against supported extensions for current OS
            supported = False
            for allowed in context.extensions:
                if filename.endswith(allowed):
                    supported = True
                    break
            if not supported:
                continue

            pkgs.append({
                "name": "nodejs",
                "version": version,
                "os": os_type,
                "arch": arch_type,
                "url": url,
                "filename": filename
            })
    return pkgs

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
