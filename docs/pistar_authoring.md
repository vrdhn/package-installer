# PiStar Authoring Guide

This guide describes how to write `pistar` recipes for the `pi` package manager. Recipes are written in [Starlark](https://github.com/bazelbuild/starlark), a dialect of Python.

## Core Concepts

A recipe file (e.g., `mypackage.star`) defines how to discover versions of a package. The primary goal is to register versions using a **Pipeline**.

### Lifecycle

1.  **Load**: The recipe is loaded and evaluated. Global statements are executed.
2.  **Registration**: You must call `add_package(name, function)` or `add_manager(name, function)` to register callbacks.
3.  **Execution**: When `pi` needs to find versions for a package, it calls the registered function.

## Global Functions

### Package Registration

*   `add_package(name, function)`: Registers a package discovery function.
    *   `name`: The package name (e.g., "node").
    *   `function`: A function that takes `(pkgname)` as an argument.

*   `add_manager(name, function)`: Registers a manager discovery function (for package managers like npm, cargo, etc.).
    *   `name`: The manager name (e.g., "npm").
    *   `function`: A function that takes `(manager, package)` as arguments.

### System Info

*   `get_os()`: Returns the operating system ("linux", "macos", "windows", etc.).
*   `get_arch()`: Returns the architecture ("x86_64", "aarch64", etc.).

### Networking

*   `download(url)`: Downloads content from `url` and returns it as a string. Caches results automatically.

### String Manipulation

*   `extract(pattern, text)`: Performs a regular expression match.
    *   Returns a list where the first element is a boolean (success).
    *   If successful, subsequent elements are the strings captured by the regex groups.
    *   If unsuccessful, subsequent elements are empty strings, ensuring safe unpacking.
    *   Example: `ok, name, version = extract(r"([a-z]+)-([0-9.]+)", "python-3.9")`

### Parsing & Querying

Pi supports parsing JSON, TOML, XML, and HTML content. It provides a structured way to query them using `Document` and `Node` objects.

#### Parsing Functions

*   `parse_json(content)`: Returns a `DataDocument`.
*   `parse_toml(content)`: Returns a `DataDocument`.
*   `parse_xml(content)`: Returns an `XmlDocument`.
*   `parse_html(content)`: Returns an `HtmlDocument`.

#### Documents and Nodes

Every document has a `.root` attribute that returns the top-level `Node`.

*   `doc.root`: Returns the root node of the document.

Nodes provide the following methods for querying and data access:

*   `node.get(key)`: Safely returns a child node or `None`.
*   `node.select(query)`: Returns a list of matching child nodes.
*   `node.select_one(query)`: Returns the first matching child node or `None`.
*   `node.attribute(name)`: Returns the value of an attribute (or key) as a string, or `None`.
*   `node.text()`: Returns the text content of the node.

---

## Pipeline API

Pi uses a pipeline-based model for installation. Instead of passing a large dictionary of fields, you create a `VersionBuilder` and define a sequence of steps.

### Creating a Version

*   `create_version(pkgname, version, release_date=None, release_type="stable")`: Returns a `VersionBuilder`.

### VersionBuilder Methods

#### Metadata
*   `v.set_stream(name)`: Sets a human-readable stream name (e.g., "Panda", "Iron").

#### Build Options (Flags)
Recipes can declare flags that users can configure in their `pi.cave.json`.

*   `v.add_flag(name, help, default)`: Declares a supported build flag.
*   `v.flag_value(name)`: Returns the resolved value of a flag (from cave config or default) as a string (e.g., `"true"`, `"false"`, or a custom string).

#### Pipeline Steps
Steps are executed in order. Each step's output (path) becomes the context for the next step.

*   `v.fetch(url, checksum=None, filename=None, name=None)`: Downloads a file.
*   `v.extract(format=None, name=None)`: Extracts the result of the previous `fetch` step.
*   `v.run(command, cwd=None, name=None)`: Runs a command in the sandbox. If `cwd` is provided, it is relative to the previous step's output.

#### Exports
Exports define how the results of the pipeline are exposed to the Cave environment.

*   `v.export_link(src, dest)`: Symlinks files from the build directory into `.pilocal`. Supports globs (e.g., `bin/*`).
*   `v.export_env(key, value)`: Sets an environment variable when the package is used.
*   `v.export_path(path)`: Adds a directory (relative to `.pilocal`) to the `PATH`.

#### Finalization
*   `v.register()`: Finalizes and registers the version defined by the builder.

---

## Examples

### Dynamic Pipeline Example (Erlang with Build Flags)

```python
def install_erlang(pkg):
    # ... discovery ...
    v = create_version("erlang", "26.0")
    
    # Declare flags
    v.add_flag("javac", "Include Java support", False)
    
    # Query flags to construct unique paths and commands
    use_javac = v.flag_value("javac")
    javac_arg = "--with-javac" if use_javac == "true" else "--without-javac"
    
    # Use flags in the install path suffix
    inst_dir = "@PACKAGES_DIR/erlang-26.0-javac-" + use_javac
    
    v.fetch(url="https://...", name="Download")
    v.extract(name="Extract")
    v.run(
        name="Build",
        command="./configure --prefix=" + inst_dir + " " + javac_arg + " && make install"
    )
    v.export_link(inst_dir + "/bin/*", "bin")
    v.register()
```

### Binary Release Example (Node.js)

```python
def install_node(pkg):
    v = create_version("node", "20.5.0")
    v.fetch("https://nodejs.org/dist/v20.5.0/node-v20.5.0-linux-x64.tar.gz")
    v.extract()
    v.export_link("node-v20.5.0-linux-x64/bin/*", "bin")
    v.register()
```
