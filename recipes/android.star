def install_android_studio(package_name):
    # Use JetBrains maintained list for easy discovery
    content = download("https://jb.gg/android-studio-releases-list.json")
    if not content:
        return
    
    doc = parse_json(content)
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
        date = item.attribute("date")
        channel = item.attribute("channel")
        
        release_type = "stable"
        if channel == "Beta":
            release_type = "testing"
        elif channel == "Canary":
            release_type = "nightly"
            
        downloads = item.select("$.download[*]")
        for dl in downloads:
            link = dl.attribute("link")
            if link and link.endswith(suffix):
                v = create_version(
                    pkgname = "android-studio",
                    version = version,
                    release_date = date,
                    release_type = release_type
                )
                filename = link.split("/")[-1]
                v.fetch(url = link, filename = filename, checksum = dl.attribute("checksum"))
                v.extract()
                v.export_link("android-studio/bin/*", "bin")
                
                add_version(v)
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
    for pkg in doc.root.select("remotePackage"):
        if pkg.attribute("path") == sdk_path:
            archives = pkg.select_one("archives")
            if not archives:
                continue
                
            for archive in archives.select("archive"):
                host_os_node = archive.select_one("host-os")
                if host_os_node and host_os_node.text() == host_os:
                    complete = archive.select_one("complete")
                    if complete:
                        url_node = complete.select_one("url")
                        checksum_node = complete.select_one("checksum")
                        
                        if url_node and checksum_node:
                            url = url_node.text()
                            checksum = checksum_node.text()
                            
                            revision = pkg.select_one("revision")
                            if revision:
                                major_node = revision.select_one("major")
                                minor_node = revision.select_one("minor")
                                if major_node and minor_node:
                                    version = major_node.text() + "." + minor_node.text()
                                    micro = revision.select_one("micro")
                                    if micro:
                                        version += "." + micro.text()
                                    
                                    v = create_version(pkgname, version)
                                    v.fetch(url = "https://dl.google.com/android/repository/" + url, filename = url, checksum = checksum)
                                    v.extract()
                                    for src, dest in filemap.items():
                                        v.export_link(src, dest)
                                    
                                    add_version(v)
                                    break

def install_android_sdk(package_name):
    discover_google_sdk("android-sdk", "cmdline-tools;latest", {"cmdline-tools/bin/*": "bin"})

def install_android_platform_tools(package_name):
    discover_google_sdk("android-platform-tools", "platform-tools", {"platform-tools/*": "bin"})

add_package("android-studio", install_android_studio)
add_package("android-sdk", install_android_sdk)
add_package("android-platform-tools", install_android_platform_tools)
