# PiStar Authoring Guide

This guide describes how to write `pistar` recipes for the `pi` package manager. Recipes are written in [Starlark](https://github.com/bazelbuild/starlark), a dialect of Python.

## Core Concepts

A recipe file (e.g., `mypackage.star`) defines how to discover versions of a package. The primary goal is to register versions using `add_version`.

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

*   `node.select(query)`: Returns a list of matching child nodes.
*   `node.select_one(query)`: Returns the first matching child node or `None`.
*   `node.attribute(name)`: Returns the value of an attribute (or key) as a string, or `None`.
*   `node.text()`: Returns the text content of the node.

**Query Syntax per Format:**

| Format | Selector Type | Reference |
| :--- | :--- | :--- |
| **JSON / TOML** | **JSONPath** | [serde_json_path](https://github.com/freestrings/serde_json_path) |
| **HTML** | **CSS Selectors** | [scraper / MDN](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors) |
| **XML** | **Tag Name** | Matches direct children by tag name |

#### Node Attributes (Additional)

*   `node.tag`: Returns the tag name (HTML/XML only).

### Version Registration

*   `add_version(...)`: Registers a discovered version.
    *   `pkgname`: Package name.
    *   `version`: Version string.
    *   `release_date`: Release date string (optional).
    *   `release_type`: Validation rules apply. Must be one of:
        *   `stable`, `testing`, `lts`, `unstable`
        *   Pattern: `major[.minor[.patch]][-suffix]` (e.g., `1.0.0`, `2.1-beta`)
    *   `url`: Download URL for the artifact.
    *   `filename`: Local filename for the artifact.
    *   `checksum`: SHA256 checksum (optional).
    *   `checksum_url`: URL to checksum file (optional).
    *   `filemap`: Dictionary mapping archive paths to install paths (e.g., `{"bin/*": "bin"}`).
    *   `env`: Dictionary of environment variables to set.
    *   `manager_command`: Command to install if using a manager (optional).

## Examples

### JSON Example (Node.js)

```python
def install_node(pkg):
    doc = parse_json(download("https://nodejs.org/dist/index.json"))
    # JSONPath query on root node
    for entry in doc.root.select("$[*]"):
        version = entry.attribute("version")
        # ... logic ...
        add_version(...)

add_package("node", install_node)
```

### HTML Example (Web Scraping)

```python
def install_app(pkg):
    doc = parse_html(download("https://example.com/downloads"))
    
    # CSS Selector on root
    for link in doc.root.select("a.download-link"):
        url = link.attribute("href")
        version = link.text()
        add_version(...)

add_package("app", install_app)
```
