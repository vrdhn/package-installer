use anyhow::Result;
use std::fs::File;
use std::io::Read;
use std::path::Path;
use sha2::{Sha256, Sha512, Digest};
use sha1::Sha1;
use hex;
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};

pub fn calculate_file_checksum(path: &Path, expected_len: usize) -> Result<String> {
    let mut file = File::open(path)?;
    let mut buffer = [0; 8192];

    match expected_len {
        40 => {
            let mut hasher = Sha1::new();
            loop {
                let n = file.read(&mut buffer)?;
                if n == 0 {
                    break;
                }
                hasher.update(&buffer[..n]);
            }
            Ok(hex::encode(hasher.finalize()))
        }
        64 => {
            let mut hasher = Sha256::new();
            loop {
                let n = file.read(&mut buffer)?;
                if n == 0 {
                    break;
                }
                hasher.update(&buffer[..n]);
            }
            Ok(hex::encode(hasher.finalize()))
        }
        128 => {
            let mut hasher = Sha512::new();
            loop {
                let n = file.read(&mut buffer)?;
                if n == 0 {
                    break;
                }
                hasher.update(&buffer[..n]);
            }
            Ok(hex::encode(hasher.finalize()))
        }
        _ => Err(anyhow::anyhow!(
            "Unsupported checksum length: {}. Expected 40 (SHA-1), 64 (SHA-256), or 128 (SHA-512).",
            expected_len
        )),
    }
}

pub fn hash_to_string<T: Hash>(val: &T) -> String {
    let mut hasher = DefaultHasher::new();
    val.hash(&mut hasher);
    format!("{:x}", hasher.finish())
}
