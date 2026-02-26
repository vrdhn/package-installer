use regex::Regex;
use std::sync::OnceLock;
use crate::models::version_entry::{ReleaseType, StructuredVersion};

static VERSION_REGEX: OnceLock<Regex> = OnceLock::new();
static COMPONENT_REGEX: OnceLock<Regex> = OnceLock::new();

pub struct InspectedVersion {
    pub version: StructuredVersion,
    pub release_type: ReleaseType,
}

pub fn inspect_version(s: &str) -> InspectedVersion {
    let version_re = VERSION_REGEX.get_or_init(|| {
        Regex::new(r"(?i)(?:v|version\s*)?(\d+(?:\.\d+)*)(?:[-._ ]?(rc|beta|alpha|nightly|canary|next|preview)(?:\.?(\d+))?)?").unwrap()
    });

    let mut release_type = ReleaseType::Stable;
    let mut components = Vec::new();
    let raw = s.to_string();

    if let Some(caps) = version_re.captures(s) {
        if let Some(v_str) = caps.get(1) {
            components = v_str.as_str()
                .split('.')
                .filter_map(|p| p.parse::<u32>().ok())
                .collect();
        }

        if let Some(rt_str) = caps.get(2) {
            let rt = rt_str.as_str().to_lowercase();
            release_type = match rt.as_str() {
                "rc" | "beta" | "alpha" | "next" | "preview" => ReleaseType::Testing,
                "nightly" | "canary" => ReleaseType::Unstable,
                _ => ReleaseType::Stable,
            };
        }
    }

    InspectedVersion {
        version: StructuredVersion {
            components,
            raw,
        },
        release_type,
    }
}
