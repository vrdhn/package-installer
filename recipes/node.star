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
    content = download("https://nodejs.org/dist/index.json")
    doc = parse_json(content)
    data = doc.root

    for i in range(len(data)):
        entry = data[i]
        version = entry["version"]

        found = False
        files = entry["files"]
        for j in range(len(files)):
            f = files[j]
            if f == platform:
                found = True
                break

        if found:
            ext = "tar.gz"
            if "win" in platform:
                ext = "zip"

            base_name = "node-" + version + "-" + platform
            filename = base_name + "." + ext
            url = "https://nodejs.org/dist/" + version + "/" + filename

            v = create_version("node")
            v.inspect(version)
            v.set_release_date(entry["date"])
            
            if entry["lts"]:
                v.set_release_type("lts")
                if entry["lts"] != "true" and entry["lts"] != True:
                    v.set_stream(entry["lts"])
            
            v.fetch(url, filename = filename)
            v.extract()
            v.export_link(base_name + "/bin/*", "bin")
            
            v.register()

add_package("node", install_node)

def npm_discovery(manager, package):
    url = "https://registry.npmjs.org/" + package
    content = download(url)
    if not content:
        return
    doc = parse_json(content)
    data = doc.root
    
    versions = data.get("versions")
    if not versions:
        return
    time = data.get("time") or {}
    dist_tags = data.get("dist-tags") or {}
    
    for version in versions:
        v = create_version(package)
        v.inspect(version)
        v.set_release_date(time.get(version, ""))
        
        if version == dist_tags.get("latest"):
            v.set_release_type("stable")
        
        v.run("npm install --prefix ~/.pilocal " + package + "@" + version)
        
        v.register()

add_manager("npm", npm_discovery)
