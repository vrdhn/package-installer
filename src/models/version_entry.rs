use crate::models::config::Config;
use crate::models::repository::Repository;
use crate::models::package_entry::{PackageEntry, ManagerEntry};
use allocative::Allocative;
use anyhow::Context as _;
use serde::{Deserialize, Serialize};
use std::fs;
use std::fmt::{self, Display};
use std::str::FromStr;
use std::sync::Arc;

/// Represents the type of a package release.
/// Example: ReleaseType::Stable
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

/// A structured representation of a version for comparison.
/// Example: { components: [1, 70, 0], raw: "1.70.0" }
#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash, Default, Eq)]
pub struct StructuredVersion {
    pub components: Vec<u32>,
    pub raw: String,
}

impl PartialOrd for StructuredVersion {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for StructuredVersion {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        for (a, b) in self.components.iter().zip(other.components.iter()) {
            if a != b {
                return a.cmp(b);
            }
        }
        self.components.len().cmp(&other.components.len())
            .then_with(|| self.raw.cmp(&other.raw))
    }
}

impl Display for StructuredVersion {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.raw)
    }
}

/// A single step in an installation pipeline.
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

/// Defines environment or file system links exported by a package.
/// Example: Export::Path("bin")
#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub enum Export {
    Link { src: String, dest: String },
    Env { key: String, val: String },
    Path(String),
}

/// A configurable flag for building the package.
#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub struct BuildFlag {
    pub name: String,
    pub help: String,
    pub default_value: String,
}

/// A dependency on another package.
#[derive(Debug, Clone, Serialize, Deserialize, Allocative, PartialEq, Hash)]
pub struct Dependency {
    pub name: String,
    pub optional: bool,
}

/// Detailed entry for a specific version of a package.
#[derive(Debug, Clone, Serialize, Deserialize, Allocative, Default)]
pub struct VersionEntry {
    /// Full name including manager prefix if any, e.g., "go:github.com/gin-gonic/gin"
    pub pkgname: String,
    pub version: StructuredVersion,
    pub release_date: String,
    pub release_type: ReleaseType,
    #[serde(default)]
    pub stream: String,
    #[serde(default)]
    pub pipeline: Vec<InstallStep>,
    #[serde(default)]
    pub exports: Vec<Export>,
    #[serde(default)]
    pub flags: Vec<BuildFlag>,
    #[serde(default)]
    pub build_dependencies: Vec<Dependency>,
}

impl VersionEntry {
    pub fn pkg_dir_name(&self) -> String {
        format!("{}-{}", crate::utils::fs::sanitize_name(&self.pkgname), crate::utils::fs::sanitize_name(&self.version.to_string()))
    }
}

/// A version entry qualified by the repository it belongs to.
#[derive(Debug, Clone)]
pub struct QualifiedVersion<'a> {
    pub repo_name: &'a str,
    pub entry: &'a VersionEntry,
}

impl<'a> QualifiedVersion<'a> {
    pub fn new(repo_name: &'a str, entry: &'a VersionEntry) -> Self {
        Self { repo_name, entry }
    }

    pub fn pkg_ctx(&self) -> String {
        format!("{}/{}={}", self.repo_name, self.entry.pkgname, self.entry.version)
    }
}

/// A collection of version entries.
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct VersionList {
    pub versions: Vec<VersionEntry>,
}

/// Options for retrieving version lists.
pub struct GetVersionOptions<'a> {
    pub config: &'a Config,
    pub repo: &'a Repository,
    pub package_name: &'a str,
    pub package_entry: Option<&'a PackageEntry>,
    pub manager_entry: Option<(&'a ManagerEntry, &'a str)>,
    pub force: bool,
}

impl VersionList {
    /// Retrieves the version list for a package, using cache if available.
    pub fn get_for_package(opts: GetVersionOptions) -> Option<Arc<Self>> {
        let key = format!("{}:{}", opts.repo.name, opts.package_name);
        use dashmap::mapref::entry::Entry;

        // Check cache first using DashMap for thread-safe concurrent access.
        if !opts.config.force && !opts.force {
            if let Entry::Occupied(occupied) = opts.config.state.version_lists.entry(key.clone()) {
                let arc_list: Arc<VersionList> = occupied.get().clone();
                return Some(arc_list);
            }
        }

        if let Some(list) = try_load_from_disk(opts.config, opts.repo, opts.package_name, opts.force, &key) {
            return Some(list);
        }

        sync_and_load(opts, &key)
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
        fs::create_dir_all(&config.cache_meta_dir).context("Failed to create meta directory")?;
        let safe_name = package_name.replace('/', "#");
        let cache_file = config.version_cache_file(repo_name, &safe_name);
        let content =
            serde_json::to_string_pretty(self).context("Failed to serialize version list")?;
        fs::write(&cache_file, content)
            .with_context(|| format!("Failed to write version cache file: {:?}", cache_file))
    }
}

fn try_load_from_disk(config: &Config, repo: &Repository, name: &str, force_opt: bool, key: &str) -> Option<Arc<VersionList>> {
    if !config.force && !force_opt {
        if let Ok(list) = VersionList::load(config, &repo.name, name) {
            let arc_list = Arc::new(list);
            config.state.version_lists.insert(key.to_string(), arc_list.clone());
            return Some(arc_list);
        }
    }
    None
}

fn sync_and_load(opts: GetVersionOptions, key: &str) -> Option<Arc<VersionList>> {
    if let Some(pkg) = opts.package_entry {
        if let Err(e) = crate::services::sync::sync_package(opts.config, opts.repo, pkg) {
            log::error!("[{}/{}] sync failed: {}", opts.repo.name, pkg.name, e);
        }
    } else if let Some((mgr, pkg_name)) = opts.manager_entry {
        let manager_name = opts.package_name.split(':').next().unwrap_or("");
        if let Err(e) = crate::services::sync::sync_manager_package(
            opts.config,
            opts.repo,
            mgr,
            manager_name,
            pkg_name,
        ) {
            log::error!("[{}/{}:{}] sync failed: {}", opts.repo.name, manager_name, pkg_name, e);
        }
    }

    if let Ok(list) = VersionList::load(opts.config, &opts.repo.name, opts.package_name) {
        let arc_list = Arc::new(list);
        opts.config.state.version_lists.insert(key.to_string(), arc_list.clone());
        return Some(arc_list);
    }
    None
}
