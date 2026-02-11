def install_node(package_name):
    print("Installing Node:", package_name)
    
    # Download nodejs versions json
    print("Downloading nodejs versions...")
    content = download("https://nodejs.org/dist/index.json")
    
    # Parse json
    data = json_parse(content)
    
    # Dump latest version using JSONPath
    print("Latest NodeJS version info:")
    json_dump(data, "$[0]")

def install_firefox(package_name):
    print("Installing Firefox:", package_name)

add_package("^node", install_node)
add_package("^firefox", install_firefox)