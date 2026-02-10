mod devel_test;
mod starlark_executor;
mod config;

use clap::{Parser, Subcommand};
use log::LevelFilter;

#[derive(Parser)]
#[command(name = "pi")]
#[command(about = "A package installer", long_about = None)]
struct Cli {
    /// Enable verbose logging (shows all log levels)
    #[arg(short, long, global = true)]
    verbose: bool,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Development commands
    Devel {
        #[command(subcommand)]
        command: DevelCommands,
    },
}

#[derive(Subcommand)]
enum DevelCommands {
    /// Test a package
    Test {
        /// The filename to test
        filename: String,
        /// Optional package name
        pkg: Option<String>,
    },
}

fn main() {
    let cli = Cli::parse();

    let log_level = if cli.verbose {
        LevelFilter::Trace
    } else {
        LevelFilter::Info
    };

    env_logger::Builder::new()
        .filter_level(log_level)
        .init();

    match &cli.command {
        Commands::Devel { command } => match command {
            DevelCommands::Test { filename, pkg } => {
                devel_test::run(filename, pkg.as_deref());
            }
        },
    }
}