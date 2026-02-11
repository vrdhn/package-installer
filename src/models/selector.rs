use regex::Regex;

#[derive(Debug, Clone)]
pub struct PackageSelector {
    pub recipe: Option<String>,
    pub prefix: Option<String>,
    pub package: String,
    pub version: Option<String>,
}

impl PackageSelector {
    /// Parses a selector string in the format: [recipe]/[prefix]:package[=version]
    pub fn parse(s: &str) -> Option<Self> {
        // Updated regex to handle [recipe/][prefix:]package
        // 1. Optional recipe followed by /
        // 2. Optional prefix followed by :
        // 3. Package name (can contain /)
        // 4. Optional version starting with =

        let re = Regex::new(r"^(?:([^/:]+)/)?(?:([^/:]+):)?([^=]+)(?:=(.+))?$").unwrap();
        let caps = re.captures(s)?;

        Some(Self {
            recipe: caps.get(1).map(|m| m.as_str().to_string()),
            prefix: caps.get(2).map(|m| m.as_str().to_string()),
            package: caps.get(3).map(|m| m.as_str().to_string()).unwrap(),
            version: caps.get(4).map(|m| m.as_str().to_string()),
        })
    }
}
