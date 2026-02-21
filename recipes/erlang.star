def get_platform():
    return get_os(), get_arch()

def install_erlang(package_name):
    # Erlang/OTP releases from GitHub
    # We'll fetch the latest releases
    content = download("https://api.github.com/repos/erlang/otp/releases")
    if not content:
        return
    releases_doc = parse_json(content)
    releases = releases_doc.root

    for i in range(len(releases)):
        release = releases[i]
        tag = release["tag_name"]
        if not tag.startswith("OTP-"):
            continue
        
        version = tag[4:]
        # Filter out R versions (very old)
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
            # Fallback to source code tarball from GitHub if no asset
            url = "https://github.com/erlang/otp/archive/refs/tags/" + tag + ".tar.gz"
            filename = "otp-" + version + ".tar.gz"

        # Build from source
        # We use a local prefix to avoid polluting the sandbox home during build
        # and then use filemap to link it into .pilocal
        build_cmd = "./otp_build autoconf && ./configure --prefix=$(pwd)/_inst --without-termcap && make -j$(nproc) && make install"
        
        # Note: some versions might not have otp_build autoconf if they are already configured
        # but GitHub source tarballs usually need it.
        # Pre-packaged release assets usually don't need autoconf.

        v = create_version(
            pkgname = "erlang",
            version = version,
            release_date = release["published_at"],
            release_type = "stable"
        )
        v.fetch(url = url, filename = filename)
        v.extract()
        v.run("if [ -f otp_build ]; then ./otp_build autoconf; fi && ./configure --prefix=$(pwd)/_inst --without-termcap && make -j$(nproc) && make install")
        v.export_link("_inst/bin/*", "bin")
        v.export_link("_inst/lib/erlang/*", "lib/erlang")
        
        add_version(v)

add_package("erlang", install_erlang)
