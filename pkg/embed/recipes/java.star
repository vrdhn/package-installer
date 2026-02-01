def discover(version_query):
    return {
        "url": "https://api.foojay.io/disco/v3.0/packages?package_type=jdk&release_status=ga",
        "method": "GET"
    }

def parse(data, version_query):
    resp = json.decode(data)
    pkgs = []
    
    for p in resp["result"]:
        version = p["java_version"]
        if version_query != "latest" and version_query != "" and not version.startswith(version_query):
            continue
            
        pkgs.append({
            "name": "java-" + p["distribution"],
            "version": version,
            "os": p["operating_system"],
            "arch": p["architecture"],
            "url": p["links"]["pkg_download_redirect"],
            "filename": p["filename"],
            "checksum": p.get("checksum", "")
        })
    return pkgs
