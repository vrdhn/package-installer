def discover(pkg_name, version_query, context):
    # Fetch all GA JDKs
    url = "https://api.foojay.io/disco/v3.0/packages?package_type=jdk&release_status=ga"
    
    return {
        "url": url,
        "method": "GET"
    }

def parse(pkg_name, data, version_query, context):
    resp = json.decode(data)
    pkgs = []
    
    for p in resp["result"]:
        version = p["java_version"]
        
        status = p.get("release_status", "unknown")
        if status == "ga":
            status = "stable"
            
        filename = p["filename"]
        # Removed extension filtering

        # Map OS names if necessary
        os_raw = p["operating_system"]
        os_type = os_raw
        if os_raw == "macos":
            os_type = "darwin"
        
        # Map Arch names if necessary
        # Foojay uses aarch64, x64 etc. which pi might expect to normalize
        # pi expects: x64, arm64.
        # Foojay returns: x64, aarch64, x86, arm32 etc.
        arch_raw = p["architecture"]
        arch_type = arch_raw
        if arch_raw == "aarch64":
            arch_type = "arm64"
        elif arch_raw == "amd64": # Just in case
            arch_type = "x64"

        pkgs.append({
            "name": "java-" + p["distribution"],
            "version": version,
            "release_status": status,
            "os": os_type,
            "arch": arch_type,
            "url": p["links"]["pkg_download_redirect"],
            "filename": filename,
            "checksum": p.get("checksum", "")
        })
    return pkgs
