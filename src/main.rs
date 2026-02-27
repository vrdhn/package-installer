mod build;
mod cli;
mod commands;
mod logging;
mod models;
mod services;
mod starlark;
mod utils;

use crate::cli::parser::{Cli, Commands, DevelCommands, CaveCommands, RepoCommands, PackageCommands, DiskCommands};
use crate::logging::init::init_logging;
use crate::models::config::Config;
use clap::Parser;

fn main() {
    let cli = Cli::parse();
    let config = Config::new(cli.force);

    init_logging(cli.quiet, cli.verbose, cli.debug);

    if config.is_inside_cave() {
        validate_command_in_cave(&cli.command);
    }

    route_command(cli.command, &config);
}

/// Validates that the command is allowed to run when PI_CAVE is set.
fn validate_command_in_cave(command: &Commands) {
    let is_allowed = match command {
        Commands::Version | 
        Commands::Repo { command: RepoCommands::List { .. } } |
        Commands::Package { command: PackageCommands::List { .. } } |
        Commands::Package { command: PackageCommands::Info { .. } } |
        Commands::Package { command: PackageCommands::Resolve { .. } } |
        Commands::Cave { command: CaveCommands::Info } => true,
        _ => false,
    };

    if !is_allowed {
        log::error!("command not allowed inside cave");
        std::process::exit(1);
    }
}

/// Routes the CLI command to the appropriate handler.
fn route_command(command: Commands, config: &Config) {
    match command {
        Commands::Version => {
            println!("v{}", build::BUILD_VERSION);
            println!("build {}", build::BUILD_DATE);
        }
        Commands::Repo { command } => handle_repo_command(command, config),
        Commands::Package { command } => handle_package_command(command, config),
        Commands::Cave { command } => handle_cave_command(command, config),
        Commands::Disk { command } => handle_disk_command(command, config),
        Commands::Devel { command } => handle_devel_command(command, config),
    }
}

fn handle_repo_command(command: RepoCommands, config: &Config) {
    match command {
        RepoCommands::Add { path } => commands::repo::add::run(config, &path),
        RepoCommands::Sync { name } => commands::repo::sync::run(config, name.as_deref()),
        RepoCommands::List { name } => commands::repo::list::run(config, name.as_deref()),
    }
}

fn handle_package_command(command: PackageCommands, config: &Config) {
    match command {
        PackageCommands::Sync { selector } => commands::package::sync::run(config, selector.as_deref()),
        PackageCommands::List { selector, all } => commands::package::list::run(config, selector.as_deref(), all),
        PackageCommands::Info { selector } => commands::package::info::run(config, &selector),
        PackageCommands::Resolve { queries } => commands::package::resolve::run(config, queries),
    }
}

fn handle_cave_command(command: CaveCommands, config: &Config) {
    match command {
        CaveCommands::Init => commands::cave::init::run(config),
        CaveCommands::Info => commands::cave::info::run(config),
        CaveCommands::Add { args } => commands::cave::add::run(config, args),
        CaveCommands::Rem { args } => commands::cave::rem::run(config, args),
        CaveCommands::Resolve { variant } => commands::cave::resolve::run(config, variant),
        CaveCommands::Build { variant } => commands::cave::build::run(config, variant),
        CaveCommands::Run { variant, command } => commands::cave::run::run(config, variant, command),
    }
}

fn handle_disk_command(command: DiskCommands, config: &Config) {
    match command {
        DiskCommands::Info => commands::disk::info::run(config),
        DiskCommands::Clean => commands::disk::clean::run(config),
        DiskCommands::Uninstall { confirm } => commands::disk::uninstall::run(config, confirm),
    }
}

fn handle_devel_command(command: DevelCommands, config: &Config) {
    match command {
        DevelCommands::Test { filename, pkg } => commands::devel::test::run(config, &filename, pkg.as_deref()),
    }
}
