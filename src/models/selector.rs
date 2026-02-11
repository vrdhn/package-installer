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
        // Regex to capture: ([recipe]/)?([prefix]:)?package(=version)?
        // We need to be careful with the separators.
        
        let re = Regex::new(r"^(?:([^/]+)/)?(?:([^:]+):)?([^=]+)(?:=(.+))?$").unwrap();
        let caps = re.captures(s)?;

        Some(Self {
            recipe: caps.get(1).map(|m| m.as_str().to_string()),
            prefix: caps.get(2).map(|m| m.as_str().to_string()),
            package: caps.get(3).map(|m| m.as_str().to_string()).unwrap(),
            version: caps.get(4).map(|m| m.as_str().to_string()),
        })
    }
}
