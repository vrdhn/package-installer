def discover(pkg_name, version_query, context):
    # Map pi OS names to foojay OS names
    os_map = {
        "linux": "linux",
        "darwin": "macos",
        "windows": "windows"
    }
    foojay_os = os_map.get(context.os, context.os)
    
    # Preferred extension
    ext = "tar.gz"
    if context.os == "windows":
        ext = "zip"
        
    url = "https://api.foojay.io/disco/v3.0/packages?package_type=jdk&release_status=ga&operating_system=%s&archive_type=%s" % (foojay_os, ext)
    
    return {
        "url": url,
        "method": "GET"
    }

def parse(pkg_name, data, version_query, context):
    resp = json.decode(data)
    pkgs = []
    
    for p in resp["result"]:
        version = p["java_version"]
        if version_query != "latest" and version_query != "" and not version.startswith(version_query):
            continue
            
        # Ensure we only take supported extensions
        filename = p["filename"]
        supported = False
        for ext in context.extensions:
            if filename.endswith(ext):
                supported = True
                break
        if not supported:
            continue

        pkgs.append({
            "name": "java-" + p["distribution"],
            "version": version,
            "os": p["operating_system"],
            "arch": p["architecture"],
            "url": p["links"]["pkg_download_redirect"],
            "filename": filename,
            "checksum": p.get("checksum", "")
        })
    return pkgs
