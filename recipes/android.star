def install_android_studio(package_name):
    # Use JetBrains maintained list for easy discovery
    content = download("https://jb.gg/android-studio-releases-list.json")
    if not content:
        return
    
    doc = parse_json(content)
    # The JSON structure has a "content" field which contains "item" list
    # Use select/select_one on the document or root
    items = doc.root.select("$.content.item[*]")
    
    os_name = get_os()
    arch = get_arch()
    
    suffix = ""
    if os_name == "linux":
        suffix = "-linux.tar.gz"
    elif os_name == "macos":
        if arch == "aarch64":
            suffix = "-mac_arm.dmg"
        else:
            suffix = "-mac.dmg"
    elif os_name == "windows":
        suffix = "-windows.zip"
        
    for item in items:
        version = item.attribute("version")
        name = item.attribute("name")
        build = item.attribute("build")
        date = item.attribute("date")
        channel = item.attribute("channel")
        
        release_type = "stable"
        if channel == "Beta":
            release_type = "testing"
        elif channel == "Canary":
            release_type = "nightly"
            
        # For nested lists in JSON, select can be used
        downloads = item.select("$.download[*]")
        for dl in downloads:
            link = dl.attribute("link")
            if link and link.endswith(suffix):
                add_version(
                    pkgname = "android-studio",
                    version = version,
                    release_date = date,
                    release_type = release_type,
                    url = link,
                    filename = link.split("/")[-1],
                    checksum = dl.attribute("checksum") or "",
                    checksum_url = "",
                    filemap = {"android-studio/bin/*": "bin"}
                )
                break

def discover_google_sdk(pkgname, sdk_path, filemap):
    content = download("https://dl.google.com/android/repository/repository2-1.xml")
    if not content:
        return
        
    os_name = get_os()
    host_os = "linux"
    if os_name == "macos":
        host_os = "macosx"
    elif os_name == "windows":
        host_os = "windows"

    doc = parse_xml(content)
    # select / select_one are now methods on the parsed objects
    for pkg in doc.root.select("remotePackage"):
        if pkg.attribute("path") == sdk_path:
            archives = pkg.select_one("archives")
            if not archives:
                continue
                
            for archive in archives.select("archive"):
                host_os_node = archive.select_one("host-os")
                if host_os_node and host_os_node.text() == host_os:
                    complete = archive.select_one("complete")
                    if not complete:
                        continue

                    url_node = complete.select_one("url")
                    checksum_node = complete.select_one("checksum")
                    
                    if not url_node or not checksum_node:
                        continue

                    url = url_node.text()
                    checksum = checksum_node.text()
                    
                    revision = pkg.select_one("revision")
                    if not revision:
                        continue

                    major_node = revision.select_one("major")
                    minor_node = revision.select_one("minor")
                    if not major_node or not minor_node:
                        continue
                        
                    version = major_node.text() + "." + minor_node.text()
                    micro = revision.select_one("micro")
                    if micro:
                        version += "." + micro.text()
                    
                    add_version(
                        pkgname = pkgname,
                        version = version,
                        release_date = "",
                        release_type = "stable",
                        url = "https://dl.google.com/android/repository/" + url,
                        filename = url,
                        checksum = checksum,
                        checksum_url = "",
                        filemap = filemap
                    )
                    break
            # We found the package we wanted, but there might be multiple versions?
            # Actually Google SDK XML usually has one <remotePackage> per path.

def install_android_sdk(package_name):
    discover_google_sdk("android-sdk", "cmdline-tools;latest", {"cmdline-tools/bin/*": "bin"})

def install_android_platform_tools(package_name):
    discover_google_sdk("android-platform-tools", "platform-tools", {"platform-tools/*": "bin"})

add_package("android-studio", install_android_studio)
add_package("android-sdk", install_android_sdk)
add_package("android-platform-tools", install_android_platform_tools)
