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
    /// Repository management
    Repo {
        #[command(subcommand)]
        command: RepoCommands,
    },
    /// Development commands
    Devel {
        #[command(subcommand)]
        command: DevelCommands,
    },
}

#[derive(Subcommand)]
pub enum RepoCommands {
    /// Add a new repository
    Add {
        /// Path to the repository
        path: String,
    },
    /// Sync repositories
    Sync {
        /// Optional name of the repository to sync
        name: Option<String>,
    },
    /// List repositories and their packages
    List {
        /// Optional name of the repository to list
        name: Option<String>,
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
