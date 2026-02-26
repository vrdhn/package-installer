def get_platform():
    return get_os(), get_arch()

def install_erlang(package_name):
    # Erlang/OTP releases from GitHub
    content = download("https://api.github.com/repos/erlang/otp/releases")
    if not content:
        return
    releases_doc = parse_json(content)
    releases = releases_doc.root

    for i in range(len(releases)):
        release = releases[i]
        tag = release["tag_name"]
        ok, version = extract(r"OTP-([0-9.]+.*)", tag)
        if not ok:
            continue
        
        if version.startswith("R"):
            continue

        # Look for source tarball
        url = ""
        filename = ""
        assets = release["assets"]
        for j in range(len(assets)):
            asset = assets[j]
            name = asset["name"]
            if name.endswith(".tar.gz") and "patch" not in name and "_doc_" not in name and "_src_" in name:
                url = asset["browser_download_url"]
                filename = name
                break
        
        if not url:
            url = "https://github.com/erlang/otp/archive/refs/tags/" + tag + ".tar.gz"
            filename = "otp-" + version + ".tar.gz"

        v = create_version("erlang")
        v.inspect(version)
        v.set_release_date(release["published_at"])
        
        v.add_flag(name="javac", help="Include Java support", default=False)
        v.add_flag(name="termcap", help="Include termcap support", default=False)

        # Resolve flags to strings for path/cmd construction
        javac_val = v.flag_value("javac")
        termcap_val = v.flag_value("termcap")
        
        javac_flag = "--with-javac" if javac_val == "true" else "--without-javac"
        termcap_flag = "--with-termcap" if termcap_val == "true" else "--without-termcap"
        
        # Suffix includes flags state
        suffix = "javac-" + javac_val + "-termcap-" + termcap_val
        inst_dir = "@PACKAGES_DIR/erlang-" + version + "-" + suffix

        v.fetch(url = url, filename = filename, name = "Download Source")
        v.extract(name = "Extract Source")
        
        ok_ext, src_dir = extract(r"(.*)\.tar\.gz", filename)
        if not ok_ext:
            src_dir = filename
        
        v.run(
            name = "Compile and Install",
            command = "if [ -f otp_build ]; then ./otp_build autoconf; fi && ./configure --prefix=" + inst_dir + " " + javac_flag + " " + termcap_flag + " && make -j$(nproc) && make install",
            cwd = src_dir
        )
        
        v.export_link(inst_dir + "/bin/*", "bin")
        v.export_link(inst_dir + "/lib/erlang/*", "lib/erlang")
        
        v.register()

def erlang_discovery(manager, package):
    if package != "rebar3":
        return

    content = download("https://api.github.com/repos/erlang/rebar3/releases")
    if not content:
        return
    releases_doc = parse_json(content)
    releases = releases_doc.root

    for i in range(len(releases)):
        release = releases[i]
        tag = release["tag_name"]
        version = tag
        
        # Source tarball
        url = "https://github.com/erlang/rebar3/archive/refs/tags/" + tag + ".tar.gz"
        filename = "rebar3-" + version + ".tar.gz"

        v = create_version("erlang:rebar3")
        v.inspect(version)
        v.set_release_date(release["published_at"])
        
        # rebar3 requires erlang to build
        v.require("erlang")
        
        v.fetch(url = url, filename = filename, name = "Download Source")
        v.extract(name = "Extract Source")
        
        ok_ext, src_dir = extract(r"(.*)\.tar\.gz", filename)
        if not ok_ext:
            src_dir = filename
        
        v.run(
            name = "Bootstrap",
            command = "./bootstrap",
            cwd = src_dir
        )
        
        v.export_link(src_dir + "/rebar3", "bin/rebar3")
        
        v.register()

add_package("erlang", install_erlang)
add_manager("erlang", erlang_discovery)
