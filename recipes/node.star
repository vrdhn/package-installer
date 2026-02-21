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

            release_type = "stable"
            if entry["lts"]:
                release_type = "lts"

            v = create_version("node", version, release_date = entry["date"], release_type = release_type)
            v.fetch(url, filename = filename)
            v.extract()
            v.export_link(base_name + "/bin/*", "bin")
            
            add_version(v)

add_package("node", install_node)

def npm_discovery(manager, package):
    url = "https://registry.npmjs.org/" + package
    content = download(url)
    data = parse_json(content)
    
    versions = data["versions"]
    time = data["time"]
    dist_tags = data["dist-tags"]
    
    for version in versions:
        v_data = versions[version]
        
        release_type = "stable"
        if "-" in version:
            release_type = "testing"
        
        if version == dist_tags.get("latest"):
            release_type = "stable"
        
        v = create_version(package, version, release_date = time.get(version, ""), release_type = release_type)
        v.run("npm install --prefix ~/.pilocal " + package + "@" + version)
        
        add_version(v)

add_manager("npm", npm_discovery)
