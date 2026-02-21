pub fn match_version_with_wildcard(version: &str, pattern: &str) -> bool {
    let version_parts: Vec<&str> = version.split('.').collect();
    let pattern_parts: Vec<&str> = pattern.split('.').collect();

    for (i, p) in pattern_parts.iter().enumerate() {
        if *p == "*" {
            // First * means match the rest
            return version_parts.len() >= i;
        }

        if i >= version_parts.len() || version_parts[i] != *p {
            return false;
        }
    }

    version_parts.len() == pattern_parts.len()
}
