def install_elixir_ls(package_name):
    # Fetch from GitHub releases
    content = download("https://api.github.com/repos/elixir-lsp/elixir-ls/releases")
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

        # Source tarball URL
        url = "https://github.com/elixir-lsp/elixir-ls/archive/" + tag + ".tar.gz"
        filename = "elixir-ls-" + version + ".tar.gz"

        v = create_version("elixir-ls")
        v.inspect(version)
        v.set_release_date(release["published_at"])

        # ElixirLS needs Elixir and Erlang to build and run
        v.require("elixir")
        v.require("erlang")

        v.fetch(url = url, filename = filename)
        v.extract()

        # The PKGBUILD uses mix to build
        # We'll use a local mix-cache to avoid polluting the host
        # And release to a specific directory
        inst_dir = "@PACKAGES_DIR/elixir-ls-" + version

        src_dir = "elixir-ls-" + version

        v.run(
            name = "Build",
            command = "export MIX_ENV=prod && export MIX_HOME=$PWD/mix-cache && mix local.hex --force && mix local.rebar --force && mix deps.get && mix compile && mix elixir_ls.release2 -o " + inst_dir,
            cwd = src_dir
        )

        # Expose the language server scripts from the release directory
        v.export_link(inst_dir + "/language_server.sh", "bin/elixir-ls")
        v.export_link(inst_dir + "/launch.sh", "bin/elixir-ls-launch")
        v.export_link(inst_dir + "/debugger.sh", "bin/elixir-ls-debug")

        v.register()

add_package("elixir-ls", install_elixir_ls)
