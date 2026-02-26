def install_elixir(package_name):
    # Fetch from GitHub releases
    content = download("https://api.github.com/repos/elixir-lang/elixir/releases")
    if not content:
        return
    releases_doc = parse_json(content)
    releases = releases_doc.root

    for i in range(len(releases)):
        release = releases[i]
        tag = release["tag_name"]
        ok, version = extract(r"v?([0-9.]+.*)", tag)
        if not ok:
            version = tag
        
        assets = release["assets"]
        for j in range(len(assets)):
            asset = assets[j]
            name = asset["name"]
            
            ok_otp, otp_ver = extract(r"elixir-otp-([^.]+)\.zip", name)
            if ok_otp:
                v = create_version("elixir")
                v.inspect(version + "-otp-" + otp_ver)
                v.set_release_date(release["published_at"])
                
                v.require("erlang=" + otp_ver + ".*")
                
                v.fetch(url = asset["browser_download_url"], filename = name)
                v.extract()
                v.export_link("bin/*", "bin")
                v.export_link("lib/*", "lib")
                
                v.register()
            elif name == "Precompiled.zip":
                # Some releases use this name
                v = create_version("elixir")
                v.inspect(version)
                v.set_release_date(release["published_at"])
                v.fetch(url = asset["browser_download_url"], filename = "elixir-" + version + ".zip")
                v.extract()
                v.export_link("bin/*", "bin")
                v.export_link("lib/*", "lib")
                
                v.register()

add_package("elixir", install_elixir)

def hex_discovery(manager, package):
    # Hex is usually just a command to install via mix
    if package == "hex":
        v = create_version(pkgname = "hex", version = "latest")
        v.run("mix local.hex --force && mix local.rebar --force")
        v.register()

add_manager("hex", hex_discovery)
