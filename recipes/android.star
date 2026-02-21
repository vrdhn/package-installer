def get_android_studio_suffix():
    os_name = get_os()
    arch = get_arch()
    
    if os_name == "linux":
        return "-linux.tar.gz"
    elif os_name == "macos":
        if arch == "aarch64":
            return "-mac_arm.dmg"
        else:
            return "-mac.dmg"
    elif os_name == "windows":
        return "-windows.zip"
    return ""

def format_jb_date(date_str):
    if not date_str:
        return ""
    # Format: "February 13, 2026"
    parts = date_str.replace(",", "").split(" ")
    if len(parts) != 3:
        return date_str
        
    month_map = {
        "January": "01", "February": "02", "March": "03", "April": "04",
        "May": "05", "June": "06", "July": "07", "August": "08",
        "September": "09", "October": "10", "November": "11", "December": "12"
    }
    
    m = month_map.get(parts[0], "00")
    day = parts[1]
    if len(day) == 1:
        day = "0" + day
    year = parts[2]
    
    return year + "-" + m + "-" + day

def install_android_studio(package_name):
    # JetBrains list is the most comprehensive for URLs and versions
    content = download("https://jb.gg/android-studio-releases-list.json")
    if not content:
        install_android_studio_official(package_name)
        return
    
    doc = parse_json(content)
    items = doc.root.select("$.content.item[*]")
    suffix = get_android_studio_suffix()
        
    for i in range(len(items)):
        item = items[i]
        version = item.attribute("version")
        date = format_jb_date(item.attribute("date"))
        channel = item.attribute("channel")
        name = item.attribute("name") or ""
        
        # Extract stream from name: "Android Studio Panda 1 | ..." -> "Panda 1"
        stream = name.split("|")[0].replace("Android Studio", "").replace("Feature Drop", "").strip()
        
        release_type = "stable"
        if channel == "Beta" or channel == "RC":
            release_type = "testing"
        elif channel == "Canary" or channel == "Dev":
            release_type = "nightly"
        elif channel == "Release" or channel == "Patch":
            release_type = "stable"
            
        downloads = item.select("$.download[*]")
        for j in range(len(downloads)):
            dl = downloads[j]
            link = dl.attribute("link")
            if link and link.endswith(suffix):
                v = create_version(
                    pkgname = "android-studio",
                    version = version,
                    release_date = date,
                    release_type = release_type
                )
                v.set_stream(stream)
                filename = link.split("/")[-1]
                v.fetch(url = link, filename = filename, checksum = dl.attribute("checksum"))
                v.extract()
                v.export_link("android-studio/bin/*", "bin")
                
                v.register()
                break

def install_android_studio_official(package_name):
    content = download("https://developer.android.com/studio")
    if not content:
        return
        
    doc = parse_html(content)
    suffix = get_android_studio_suffix()
    
    all_links = doc.root.select("a")
    for i in range(len(all_links)):
        l = all_links[i]
        href = l.attribute("href")
        if href and href.endswith(suffix) and "/android/studio/" in href:
            filename = href.split("/")[-1]
            parts = href.split("/")
            version = "unknown"
            for p in parts:
                if p and p[0].isdigit() and "." in p:
                    version = p
                    break
            
            v = create_version("android-studio", version)
            v.fetch(url = href, filename = filename)
            v.extract()
            v.export_link("android-studio/bin/*", "bin")
            v.register()

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
    packages = doc.root.select("remotePackage")
    for i in range(len(packages)):
        pkg = packages[i]
        if pkg.attribute("path") == sdk_path:
            archives_node = pkg.select_one("archives")
            if not archives_node:
                continue
                
            archives = archives_node.select("archive")
            for j in range(len(archives)):
                archive = archives[j]
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
                                    for src in filemap:
                                        dest = filemap[src]
                                        v.export_link(src, dest)
                                    
                                    v.register()
                                    break

def install_android_sdk(package_name):
    discover_google_sdk("android-sdk", "cmdline-tools;latest", {"cmdline-tools/bin/*": "bin"})

def install_android_platform_tools(package_name):
    discover_google_sdk("android-platform-tools", "platform-tools", {"platform-tools/*": "bin"})

add_package("android-studio", install_android_studio)
add_package("android-sdk", install_android_sdk)
add_package("android-platform-tools", install_android_platform_tools)
