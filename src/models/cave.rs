use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::fs;
use anyhow::Context;

#[derive(Debug, Serialize, Deserialize, Clone, Default)]
pub struct CaveSettings {
    #[serde(default)]

    pub packages: Vec<String>,
    #[serde(default)]
    pub set: HashMap<String, String>,
    #[serde(default)]
    pub unset: Vec<String>,
    #[serde(default)]
    pub options: HashMap<String, HashMap<String, serde_json::Value>>,
}

impl CaveSettings {
    pub fn merge(&mut self, other: &CaveSettings) {
        self.packages.extend(other.packages.clone());
        self.packages.dedup();
        for (k, v) in &other.set {
            self.set.insert(k.clone(), v.clone());
        }
        for u in &other.unset {
            self.unset.push(u.clone());
            self.set.remove(u);
        }
        self.unset.dedup();
        for (pkg, opts) in &other.options {
            let target_opts = self.options.entry(pkg.clone()).or_default();
            for (k, v) in opts {
                target_opts.insert(k.clone(), v.clone());
            }
        }
    }
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Cave {
    #[serde(default)]
    pub name: String,
    pub workspace: PathBuf,
    pub homedir: PathBuf,
    #[serde(default)]
    pub settings: CaveSettings,
    #[serde(default)]
    pub variants: HashMap<String, CaveSettings>,
}

impl Cave {
    pub const FILENAME: &'static str = "pi.cave.json";

    pub fn new(path: PathBuf, homedir: PathBuf) -> Self {
        let name = path.file_name()
            .map(|n| n.to_string_lossy().into_owned())
            .unwrap_or_else(|| "default".to_string());

        Self {
            name,
            workspace: path,
            homedir,
            settings: CaveSettings::default(),
            variants: HashMap::new(),
        }
    }

    pub fn find_in_ancestry(start_path: &Path) -> Option<(PathBuf, Self)> {
        let mut current = start_path.to_path_buf();
        loop {
            let cave_file = current.join(Self::FILENAME);
            if cave_file.exists() {
                match Self::load(&cave_file) {
                    Ok(cave) => return Some((cave_file, cave)),
                    Err(e) => {
                        log::error!("failed to load cave {}: {}", cave_file.display(), e);
                    }
                }
            }
            if !current.pop() {
                break;
            }
        }
        None
    }

    pub fn load(path: &Path) -> anyhow::Result<Self> {
        let content = fs::read_to_string(path)
            .with_context(|| format!("Failed to read cave file: {:?}", path))?;
        serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse cave file: {:?}", path))
    }

    pub fn save(&self, path: &Path) -> anyhow::Result<()> {
        let content = serde_json::to_string_pretty(self)
            .context("Failed to serialize cave")?;
        fs::write(path, content)
            .with_context(|| format!("Failed to write cave file: {:?}", path))
    }

    pub fn get_effective_settings(&self, variant_name: Option<&str>) -> anyhow::Result<CaveSettings> {
        let mut settings = self.settings.clone();
        if let Some(v_name) = variant_name {
            let v_name = v_name.strip_prefix(':').unwrap_or(v_name);
            let v_settings = self.variants.get(v_name)
                .context(format!("Variant '{}' not found in cave", v_name))?;
            settings.merge(v_settings);
        }
        Ok(settings)
    }
}
