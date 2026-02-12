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
        let mut prefix = None;
        let package;
        let mut version = None;

        let rest = if let Some(idx) = s.find('=') {
            version = Some(s[idx + 1..].to_string());
            &s[..idx]
        } else {
            s
        };

        // Find recipe if present (first / before any :)
        let (rest, recipe) = if let Some(idx) = rest.find('/') {
            // Check if there's a : before this /
            if let Some(c_idx) = rest.find(':') {
                if c_idx < idx {
                    // : comes first, so no recipe, this / is part of package name
                    (rest, None)
                } else {
                    (&rest[idx + 1..], Some(rest[..idx].to_string()))
                }
            } else {
                (&rest[idx + 1..], Some(rest[..idx].to_string()))
            }
        } else {
            (rest, None)
        };

        if let Some(idx) = rest.find(':') {
            prefix = Some(rest[..idx].to_string());
            package = rest[idx + 1..].to_string();
        } else {
            package = rest.to_string();
        }

        if package.is_empty() {
            return None;
        }

        Some(Self {
            recipe,
            prefix,
            package,
            version,
        })
    }
}
