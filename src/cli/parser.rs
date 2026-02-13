use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "pi")]
#[command(about = "A package manager", long_about = None)]
#[command(arg_required_else_help = true)]
pub struct Cli {
    /// Enable verbose logging
    #[arg(short, long, global = true)]
    pub verbose: bool,

    /// Enable debug logging
    #[arg(short, long, global = true)]
    pub debug: bool,

    /// Suppress all non-error output
    #[arg(short, long, global = true)]
    pub quiet: bool,

    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Print version information
    Version,
    /// {add, sync, list}       Repository management
    Repo {
        #[command(subcommand)]
        command: RepoCommands,
    },
    /// {sync, list, resolve}   Package management
    Package {
        #[command(subcommand)]
        command: PackageCommands,
    },
    /// {init, info, add, resolve} Cave management
    Cave {
        #[command(subcommand)]
        command: CaveCommands,
    },
    /// {info, clean, uninstall} Disk management
    Disk {
        #[command(subcommand)]
        command: DiskCommands,
    },
    /// {test}                  Development commands
    Devel {
        #[command(subcommand)]
        command: DevelCommands,
    },
}

#[derive(Subcommand)]
pub enum CaveCommands {
    /// Initialize a new cave in the current directory
    Init,
    /// Display information about the current cave
    Info,
    /// Add packages to the cave or a variant
    Add {
        /// Package queries (first one can be :variant)
        #[arg(required = true)]
        args: Vec<String>,
    },
    /// Remove packages from the cave or a variant
    Rem {
        /// Package queries to remove (first one can be :variant)
        #[arg(required = true)]
        args: Vec<String>,
    },
    /// Resolve all packages in the cave or a variant
    Resolve {
        /// Optional variant name (starts with :)
        variant: Option<String>,
    },
    /// Resolve and install all packages in the cave or a variant
    Build {
        /// Optional variant name (starts with :)
        variant: Option<String>,
    },
    /// Run a command inside the cave sandbox
    Run {
        /// Optional variant name (starts with :)
        variant: Option<String>,
        /// The command to run
        #[arg(last = true)]
        command: Vec<String>,
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
pub enum PackageCommands {
    /// Sync package versions
    Sync {
        /// Package selector (without version)
        selector: Option<String>,
    },
    /// List package versions
    List {
        /// Package selector
        selector: Option<String>,
    },
    /// Display detailed information for matching packages
    Info {
        /// Package selector
        selector: String,
    },
    /// Resolve package selectors to specific versions
    Resolve {
        /// Package selectors to resolve
        #[arg(required = true)]
        queries: Vec<String>,
    },
}

#[derive(Subcommand)]
pub enum DiskCommands {
    /// Show disk usage of pi directories
    Info,
    /// Clean the cache directory
    Clean,
    /// Uninstall pi (deletes config, state, and cache)
    Uninstall {
        /// Confirmation flag to proceed with uninstallation
        #[arg(long)]
        confirm: bool,
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
