def get_platform_string():
    os = get_os()
    arch = get_arch()

    go_os = os
    if os == "windows":
        go_os = "windows"
    elif os == "macos":
        go_os = "darwin"
    elif os == "linux":
        go_os = "linux"

    go_arch = arch
    if arch == "x86_64":
        go_arch = "amd64"
    elif arch == "aarch64":
        go_arch = "arm64"
    elif arch == "i686":
        go_arch = "386"

    return go_os, go_arch

def install_go(package_name):
    go_os, go_arch = get_platform_string()
    content = download("https://go.dev/dl/?mode=json&include=all")
    doc = parse_json(content)
    data = doc.root

    for i in range(len(data)):
        entry = data[i]
        version = entry["version"]
        
        files = entry["files"]
        for j in range(len(files)):
            f = files[j]
            if f["os"] == go_os and f["arch"] == go_arch and f["kind"] == "archive":
                release_type = "stable"
                if entry["stable"] == False:
                    release_type = "testing"
                
                v = create_version("go", version, release_type = release_type)
                v.fetch("https://go.dev/dl/" + f["filename"], checksum = f["sha256"], filename = f["filename"])
                v.extract()
                v.export_link("go/bin/*", "bin")
                
                v.register()

add_package("go", install_go)

def go_discovery(manager, package):
    parts = package.split("/")
    base_url = "https://proxy.golang.org/" + package.lower()
    
    if package.startswith("golang.org/x/"):
        if len(parts) >= 3:
            package_base = "/".join(parts[:3])
            base_url = "https://proxy.golang.org/" + package_base.lower()
    elif len(parts) >= 3 and (parts[0] == "github.com" or parts[0] == "bitbucket.org"):
        package_base = "/".join(parts[:3])
        base_url = "https://proxy.golang.org/" + package_base.lower()

    content = download(base_url + "/@v/list")
    if not content:
        return

    versions = content.split("\n")
    for version in versions:
        if not version:
            continue
            
        release_type = "stable"
        if "-" in version:
            release_type = "testing"
        
        v = create_version(package, version, release_type = release_type)
        # We don't necessarily need to fetch/extract if we just run 'go install'
        # but the manager logic might expect a pipeline.
        # For 'go install', we can just use a run step.
        v.run("go install " + package + "@" + version)
        
        v.register()

add_manager("go", go_discovery)
