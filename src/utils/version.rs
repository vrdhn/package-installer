use regex::Regex;

fn part_to_regex(part: &str) -> String {
    part.split('*')
        .map(regex::escape)
        .collect::<Vec<String>>()
        .join(".*")
}

pub fn match_version_with_wildcard(version: &str, pattern: &str) -> bool {
    let mut regex_str = String::from("^");
    let parts: Vec<&str> = pattern.split('.').collect();

    for (i, part) in parts.iter().enumerate() {
        if *part == "*" {
            // If we hit a standalone *, it matches the rest
            if i == 0 {
                return true;
            }
            regex_str.push_str(r"(\..*)?");
            regex_str.push('$');
            break;
        }

        if i > 0 {
            regex_str.push_str(r"\.");
        }
        regex_str.push_str(&part_to_regex(part));

        if i == parts.len() - 1 {
            regex_str.push('$');
        }
    }

    if let Ok(re) = Regex::new(&regex_str) {
        re.is_match(version)
    } else {
        false
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_match_simple_version() {
        assert!(match_version_with_wildcard("1.15.4", "1.15.4"));
        assert!(match_version_with_wildcard("1.15.4", "1.15.*"));
        assert!(match_version_with_wildcard("1.15.4", "1.*"));
        assert!(match_version_with_wildcard("1", "1.*"));
    }

    #[test]
    fn test_match_elixir_version() {
        assert!(match_version_with_wildcard("1.15.4-otp-28", "1.*-otp-28"));
        assert!(match_version_with_wildcard("1.15.4-otp-28", "1.15.4-otp-28"));
        assert!(!match_version_with_wildcard("1.15.4-otp-27", "1.*-otp-28"));
    }
}
