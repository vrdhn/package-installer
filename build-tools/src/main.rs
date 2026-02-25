use clap::{Parser, Subcommand};
use std::collections::HashMap;
use std::fs::File;
use std::io::{BufRead, BufReader, Read, Write};
use std::process::Command;
use sha2::{Sha256, Digest};
use std::path::Path;

#[derive(Parser)]
#[command(name = "build-tools")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Covupd,
    Covdiff,
}

struct CoverageEntry {
    coverage: f64,
    checksum: String,
}

fn get_checksum(path: &Path) -> String {
    let mut file = match File::open(path) {
        Ok(f) => f,
        Err(_) => return "MISSING".to_string(),
    };
    let mut hasher = Sha256::new();
    let mut buffer = [0; 1024];
    while let Ok(n) = file.read(&mut buffer) {
        if n == 0 { break; }
        hasher.update(&buffer[..n]);
    }
    hex::encode(hasher.finalize())
}

fn run_llvm_cov() -> HashMap<String, f64> {
    println!("Running llvm-cov...");
    let output = Command::new("cargo")
        .args(&["llvm-cov", "--summary-only", "--color", "never"])
        .output()
        .expect("Failed to run cargo llvm-cov");

    if !output.status.success() {
        eprintln!("Error: Coverage generation failed.");
        std::process::exit(1);
    }

    let mut map = HashMap::new();
    let reader = BufReader::new(&output.stdout[..]);
    for line in reader.lines().flatten() {
        if line.starts_with("---") || line.starts_with("Filename") || line.trim().is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() < 10 {
            continue;
        }
        let fname = parts[0].to_string();
        let cover_idx = if parts.len() > 9 { 9 } else { parts.len() - 1 };
        if let Ok(cover) = parts[cover_idx].replace('%', "").parse::<f64>() {
            map.insert(fname, cover);
        }
    }
    map
}

fn generate_full_report() -> HashMap<String, CoverageEntry> {
    let raw_cov = run_llvm_cov();
    let mut full_report = HashMap::new();

    for (fname, coverage) in raw_cov {
        if fname == "TOTAL" {
            continue;
        }
        // Source files are in src/
        let path = Path::new("src").join(&fname);
        let checksum = get_checksum(&path);
        full_report.insert(fname, CoverageEntry { coverage, checksum });
    }
    full_report
}

fn write_custom_coverage(path: &str, report: &HashMap<String, CoverageEntry>) {
    let mut file = File::create(path).expect("Failed to create coverage file");
    let mut keys: Vec<_> = report.keys().collect();
    keys.sort();

    for k in keys {
        let entry = &report[k];
        writeln!(file, "{} | {:.2} | {}", k, entry.coverage, entry.checksum).unwrap();
    }
}

fn parse_custom_coverage(path: &str) -> HashMap<String, CoverageEntry> {
    let mut map = HashMap::new();
    if let Ok(file) = File::open(path) {
        let reader = BufReader::new(file);
        for line in reader.lines().flatten() {
            let parts: Vec<&str> = line.split('|').map(|s| s.trim()).collect();
            if parts.len() == 3 {
                if let Ok(coverage) = parts[1].parse::<f64>() {
                    map.insert(parts[0].to_string(), CoverageEntry {
                        coverage,
                        checksum: parts[2].to_string(),
                    });
                }
            }
        }
    }
    map
}

fn main() {
    let cli = Cli::parse();

    match &cli.command {
        Commands::Covupd => {
            let report = generate_full_report();
            write_custom_coverage("COVERAGE.txt", &report);
            println!("COVERAGE.txt updated with checksums.");
        }
        Commands::Covdiff => {
            let old_report = parse_custom_coverage("COVERAGE.txt");
            let new_report = generate_full_report();

            let mut all_files: Vec<_> = old_report.keys().chain(new_report.keys()).cloned().collect();
            all_files.sort();
            all_files.dedup();

            let mut results = Vec::new();
            for f in all_files {
                let old_entry = old_report.get(&f);
                let new_entry = new_report.get(&f);

                let old_cov = old_entry.map(|e| e.coverage).unwrap_or(0.0);
                let old_sum = old_entry.map(|e| e.checksum.as_str()).unwrap_or("NONE");
                
                let new_cov = new_entry.map(|e| e.coverage).unwrap_or(0.0);
                let new_sum = new_entry.map(|e| e.checksum.as_str()).unwrap_or("NONE");

                let diff = new_cov - old_cov;
                let sum_changed = old_sum != new_sum;

                if diff.abs() > 0.001 || sum_changed {
                    results.push((f, old_cov, new_cov, diff, sum_changed));
                }
            }

            // Sort by Diff in increasing order
            results.sort_by(|a, b| a.3.partial_cmp(&b.3).unwrap());

            println!("\n{:<40} {:>8} {:>8} {:>8} {:>8}", "Filename", "Old %", "New %", "Diff", "SumChg");
            println!("{}", "-".repeat(77));

            for (f, old_val, new_val, diff, sum_changed) in results {
                println!(
                    "{:<40} {:>7.2}% {:>7.2}% {:>+7.2}% {:>8}",
                    f, old_val, new_val, diff, if sum_changed { "YES" } else { "no" }
                );
            }
            println!();
        }
    }
}
