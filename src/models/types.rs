use allocative::Allocative;
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};
use std::str::FromStr;

#[derive(Debug, Clone, Copy, Serialize, Deserialize, Allocative, PartialEq, Hash)]
#[serde(rename_all = "lowercase")]
pub enum OS {
    Linux,
    MacOS,
    Windows,
}

impl Display for OS {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Linux => write!(f, "linux"),
            Self::MacOS => write!(f, "macos"),
            Self::Windows => write!(f, "windows"),
        }
    }
}

impl FromStr for OS {
    type Err = anyhow::Error;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "linux" => Ok(Self::Linux),
            "macos" | "darwin" => Ok(Self::MacOS),
            "windows" => Ok(Self::Windows),
            _ => anyhow::bail!("Unknown OS: {}", s),
        }
    }
}

impl Default for OS {
    fn default() -> Self {
        #[cfg(target_os = "linux")]
        return Self::Linux;
        #[cfg(target_os = "macos")]
        return Self::MacOS;
        #[cfg(target_os = "windows")]
        return Self::Windows;
        #[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
        return Self::Linux;
    }
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, Allocative, PartialEq, Hash)]
#[serde(rename_all = "lowercase")]
pub enum Arch {
    X86_64,
    Aarch64,
    I686,
}

impl Display for Arch {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::X86_64 => write!(f, "x86_64"),
            Self::Aarch64 => write!(f, "aarch64"),
            Self::I686 => write!(f, "i686"),
        }
    }
}

impl FromStr for Arch {
    type Err = anyhow::Error;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "x86_64" | "amd64" => Ok(Self::X86_64),
            "aarch64" | "arm64" => Ok(Self::Aarch64),
            "i686" | "x86" => Ok(Self::I686),
            _ => anyhow::bail!("Unknown Arch: {}", s),
        }
    }
}

impl Default for Arch {
    fn default() -> Self {
        #[cfg(target_arch = "x86_64")]
        return Self::X86_64;
        #[cfg(target_arch = "aarch64")]
        return Self::Aarch64;
        #[cfg(target_arch = "x86")]
        return Self::I686;
        #[cfg(not(any(target_arch = "x86_64", target_arch = "aarch64", target_arch = "x86")))]
        return Self::X86_64;
    }
}
