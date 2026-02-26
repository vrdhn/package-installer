mod build;
mod cli;
mod commands;
mod logging;
mod models;
mod services;
mod starlark;
mod utils;

use crate::cli::parser::{Cli, Commands, DevelCommands, CaveCommands};
use crate::logging::init::init_logging;
use crate::models::config::Config;
use clap::Parser;

fn main() {
    let cli = Cli::parse();
    let config = Config::new();

    init_logging(cli.quiet, cli.verbose, cli.debug);

    if config.is_inside_cave() {
        match cli.command {
            Commands::Version | 
            Commands::Repo { command: cli::parser::RepoCommands::List { .. } } |
            Commands::Package { command: cli::parser::PackageCommands::List { .. } } |
            Commands::Package { command: cli::parser::PackageCommands::Info { .. } } |
            Commands::Package { command: cli::parser::PackageCommands::Resolve { .. } } |
            Commands::Cave { command: CaveCommands::Info } => {
                // Allowed commands
            }
            _ => {
                log::error!("command not allowed inside cave");
                std::process::exit(1);
            }
        }
    }

    match cli.command {
        Commands::Version => {
            println!("v{}", build::BUILD_VERSION);
            println!("build {}", build::BUILD_DATE);
        }
        Commands::Repo { command } => match command {
            cli::parser::RepoCommands::Add { path } => {
                commands::repo::add::run(&config, &path);
            }
            cli::parser::RepoCommands::Sync { name } => {
                commands::repo::sync::run(&config, name.as_deref());
            }
            cli::parser::RepoCommands::List { name } => {
                commands::repo::list::run(&config, name.as_deref());
            }
        },
        Commands::Package { command } => match command {
            cli::parser::PackageCommands::Sync { selector } => {
                commands::package::sync::run(&config, selector.as_deref());
            }
            cli::parser::PackageCommands::List { selector, all } => {
                commands::package::list::run(&config, selector.as_deref(), all);
            }
            cli::parser::PackageCommands::Info { selector } => {
                commands::package::info::run(&config, &selector);
            }
            cli::parser::PackageCommands::Resolve { queries } => {
                commands::package::resolve::run(&config, queries);
            }
        },
        Commands::Cave { command } => match command {
            CaveCommands::Init => commands::cave::init::run(&config),
            CaveCommands::Info => commands::cave::info::run(&config),
            CaveCommands::Add { args } => commands::cave::add::run(&config, args),
            CaveCommands::Rem { args } => commands::cave::rem::run(&config, args),
            CaveCommands::Resolve { variant } => commands::cave::resolve::run(&config, variant),
            CaveCommands::Build { variant } => commands::cave::build::run(&config, variant),
            CaveCommands::Run { variant, command } => commands::cave::run::run(&config, variant, command),
        },
        Commands::Disk { command } => match command {
            cli::parser::DiskCommands::Info => {
                commands::disk::info::run(&config);
            }
            cli::parser::DiskCommands::Clean => {
                commands::disk::clean::run(&config);
            }
            cli::parser::DiskCommands::Uninstall { confirm } => {
                commands::disk::uninstall::run(&config, confirm);
            }
        },
        Commands::Devel { command } => match command {
            DevelCommands::Test { filename, pkg } => {
                commands::devel::test::run(&config, &filename, pkg.as_deref());
            }
        },
    }
}
