use crate::models::config::Config;
use allocative::Allocative;
use anyhow::Context;
use serde::{Deserialize, Serialize};
use std::fs;
use std::fmt::{self, Display};
use std::str::FromStr;
use std::sync::Arc;

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash, Default)]
#[serde(rename_all = "lowercase")]
pub enum ReleaseType {
    #[default]
    Stable,
    Unstable,
    Testing,
    LTS,
}

impl Display for ReleaseType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Stable => write!(f, "stable"),
            Self::Unstable => write!(f, "unstable"),
            Self::Testing => write!(f, "testing"),
            Self::LTS => write!(f, "lts"),
        }
    }
}

impl FromStr for ReleaseType {
    type Err = anyhow::Error;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "stable" => Ok(Self::Stable),
            "unstable" => Ok(Self::Unstable),
            "testing" => Ok(Self::Testing),
            "lts" => Ok(Self::LTS),
            _ => Ok(Self::Stable),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash, Default)]
pub struct StructuredVersion {
    pub components: Vec<u32>,
    pub raw: String,
}

impl Display for StructuredVersion {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.raw)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub enum InstallStep {
    Fetch {
        name: Option<String>,
        url: String,
        checksum: Option<String>,
        filename: Option<String>,
    },
    Extract {
        name: Option<String>,
        format: Option<String>,
    },
    Run {
        name: Option<String>,
        command: String,
        cwd: Option<String>,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub enum Export {
    Link { src: String, dest: String },
    Env { key: String, val: String },
    Path(String),
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub struct BuildFlag {
    pub name: String,
    pub help: String,
    pub default_value: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub struct Dependency {
    pub name: String,
    pub optional: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize, Allocative, Default)]
pub struct VersionEntry {
    pub pkgname: String,
    pub version: StructuredVersion,
    pub release_date: String,
    pub release_type: ReleaseType,
    #[serde(default)]
    pub stream: String,
    #[serde(default, skip_serializing)]
    pub pipeline: Vec<InstallStep>,
    #[serde(default, skip_serializing)]
    pub exports: Vec<Export>,
    #[serde(default, skip_serializing)]
    pub flags: Vec<BuildFlag>,
    #[serde(default, skip_serializing)]
    pub build_dependencies: Vec<Dependency>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct VersionList {
    pub versions: Vec<VersionEntry>,
}

impl VersionList {
    pub fn get_for_package(
        config: &Config,
        repo: &crate::models::repository::Repository,
        package_name: &str,
        package_entry: Option<&crate::models::package_entry::PackageEntry>,
        manager_entry: Option<(&crate::models::package_entry::ManagerEntry, &str)>,
    ) -> Option<Arc<Self>> {
        let key = format!("{}:{}", repo.name, package_name);
        use dashmap::mapref::entry::Entry;

        match config.state.version_lists.entry(key) {
            Entry::Occupied(occupied) => Some(occupied.get().clone()),
            Entry::Vacant(vacant) => {
                if let Ok(list) = Self::load(config, &repo.name, package_name) {
                    let arc_list = Arc::new(list);
                    return Some(vacant.insert(arc_list).clone());
                }

                if let Some(pkg) = package_entry {
                    crate::services::sync::sync_package(config, repo, pkg);
                } else if let Some((mgr, pkg_name)) = manager_entry {
                    crate::services::sync::sync_manager_package(
                        config,
                        repo,
                        mgr,
                        package_name.split(':').next().unwrap_or(""),
                        pkg_name,
                    );
                }

                if let Ok(list) = Self::load(config, &repo.name, package_name) {
                    let arc_list = Arc::new(list);
                    return Some(vacant.insert(arc_list).clone());
                }
                None
            }
        }
    }

    pub fn load(config: &Config, repo_name: &str, package_name: &str) -> anyhow::Result<Self> {
        let safe_name = package_name.replace('/', "#");
        let cache_file = config.version_cache_file(repo_name, &safe_name);
        let content = fs::read_to_string(&cache_file)
            .with_context(|| format!("Failed to read version cache file: {:?}", cache_file))?;
        serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse version cache file: {:?}", cache_file))
    }

    pub fn save(&self, config: &Config, repo_name: &str, package_name: &str) -> anyhow::Result<()> {
        fs::create_dir_all(&config.meta_dir).context("Failed to create meta directory")?;
        let safe_name = package_name.replace('/', "#");
        let cache_file = config.version_cache_file(repo_name, &safe_name);
        let content =
            serde_json::to_string_pretty(self).context("Failed to serialize version list")?;
        fs::write(&cache_file, content)
            .with_context(|| format!("Failed to write version cache file: {:?}", cache_file))
    }
}
