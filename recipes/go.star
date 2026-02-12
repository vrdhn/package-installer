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
    print("Detected platform for Go:", go_os, go_arch)

    # Go releases API
    content = download("https://go.dev/dl/?mode=json&include=all")
    data = json_parse(content)

    for i in range(len(data)):
        entry = data[i]
        version = entry["version"] # e.g. "go1.22.0"
        
        # Look for a matching file for this version
        for f in entry["files"]:
            if f["os"] == go_os and f["arch"] == go_arch and f["kind"] == "archive":
                release_type = "stable"
                if entry["stable"] == False:
                    # Likely a beta or rc
                    release_type = "testing"
                
                # Heuristic: older versions are "obsolete" if they aren't the latest stable
                # but we'll stick to stable/testing for now as per node.star logic

                add_version(
                    pkgname = "go",
                    version = version,
                    release_date = "", # Go JSON doesn't provide release date per version directly in this format
                    release_type = release_type,
                    url = "https://go.dev/dl/" + f["filename"],
                    filename = f["filename"],
                    checksum = f["sha256"],
                    checksum_url = "",
                    filemap = {"bin/*": "bin"}
                )

add_package("go", install_go)

def go_discovery(manager, package):
    # For Go packages, we usually use the proxy to find versions
    # We'll expect package to be a full path like "golang.org/x/tools/cmd/goimports"
    print("Syncing go package:", package)
    
    parts = package.split("/")
    base_url = "https://proxy.golang.org/" + package.lower()
    
    # Heuristic for common module roots to avoid 404 on sub-packages
    if package.startswith("golang.org/x/"):
        if len(parts) >= 3:
            package_base = "/".join(parts[:3])
            base_url = "https://proxy.golang.org/" + package_base.lower()
    elif len(parts) >= 3 and (parts[0] == "github.com" or parts[0] == "bitbucket.org"):
        package_base = "/".join(parts[:3])
        base_url = "https://proxy.golang.org/" + package_base.lower()

    content = download(base_url + "/@v/list")
    if not content:
        print("No versions found for:", package)
        return

    versions = content.split("\n")
    for version in versions:
        if not version:
            continue
            
        release_type = "stable"
        if "-" in version:
            release_type = "testing"
        
        # Go install is the modern way to get binaries
        add_version(
            pkgname = package,
            version = version,
            release_date = "",
            release_type = release_type,
            url = base_url + "/@v/" + version + ".zip",
            filename = package.split("/")[-1] + "-" + version + ".zip",
            checksum = "",
            checksum_url = "",
            filemap = {"bin/*": "bin"},
            manager_command = "go install " + package + "@" + version
        )

add_manager("go", go_discovery)
