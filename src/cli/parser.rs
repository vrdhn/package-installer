use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "pi")]
#[command(about = "A package installer", long_about = None)]
pub struct Cli {
    /// Enable verbose logging (shows all log levels)
    #[arg(short, long, global = true)]
    pub verbose: bool,

    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Print version information
    Version,
    /// Development commands
    Devel {
        #[command(subcommand)]
        command: DevelCommands,
    },
}

#[derive(Subcommand)]
pub enum DevelCommands {
    /// Test a package
    Test {
        /// The filename to test
        filename: String,
        /// Optional package name
        pkg: Option<String>,
    },
}
