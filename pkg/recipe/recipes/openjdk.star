def discover(pkg_name, version_query, context):
    return {
        "url": "https://jdk.java.net/archive/",
        "method": "GET"
    }

def parse(pkg_name, data, version_query, context):
    print("Starting OpenJDK parse...")
    
    # Convert HTML to a JSON-compatible tree
    full_tree = html.to_json(data)
    # print(json.encode(full_tree))
    
    # Use JQ to find all rows in the 'builds' table
    # We use .. to find tr anywhere inside the table (handles tbody)
    rows_query = '.. | select(.tag? == "table" and .attr?.class == "builds") | .. | select(.tag? == "tr")'
    rows = jq.query(rows_query, full_tree)
    
    if not rows:
        # Fallback: just find ANY table with many rows
        rows = jq.query('.. | select(.tag? == "table") | select((.children | length) > 10) | .children[]? | select(.tag == "tr")', full_tree)
        if not rows:
            return []

    if type(rows) != "list":
        rows = [rows]

    pkgs = []
    current_version = ""
    
    for row in rows:
        # Check if row is a version header (usually has one <th>)
        headers = jq.query('.children[]? | select(.tag == "th")', row)
        if not headers:
            headers = []
        elif type(headers) != "list":
            headers = [headers]

        if len(headers) == 1:
            txt = headers[0]["text"].strip()
            if txt and not any([x in txt.lower() for x in ["linux", "windows", "macos", "os/arch"]]):
                current_version = txt.split(" ")[0]
                continue

        if not current_version:
            continue

        # Removed version filtering

        links = jq.query('.. | select(.tag? == "a" and (.attr?.href? | startswith("http")))', row)
        if not links:
            continue
        if type(links) != "list":
            links = [links]

        for link in links:
            href = link["attr"]["href"]
            filename = href.split("/")[-1].lower()
            
            if filename.endswith(".sha256") or filename.endswith(".sig"):
                continue
                
            url_os = "unknown"
            if "linux" in filename: url_os = "linux"
            elif "macos" in filename or "osx" in filename: url_os = "darwin"
            elif "windows" in filename or "win" in filename: url_os = "windows"
            
            # Removed OS filtering
                
            url_arch = "unknown"
            if "x64" in filename or "x86_64" in filename or "amd64" in filename:
                url_arch = "x64"
            elif "aarch64" in filename or "arm64" in filename:
                url_arch = "arm64"
            
            # Removed Arch filtering
                
            # Removed extension filtering
            
            status = "stable"
            if "ea" in version.lower() or "ea" in filename:
                status = "ea"
            
            pkgs.append({
                "name": "openjdk",
                "version": current_version,
                "release_status": status,
                "os": url_os,
                "arch": url_arch,
                "url": href,
                "filename": filename,
                "symlinks": {
                    "bin/*": ".local/bin"
                }
            })
    
    print("Found {} matching packages".format(len(pkgs)))
    return pkgs
