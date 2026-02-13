mod build;
mod cli;
mod commands;
mod logging;
mod models;
mod services;
mod starlark;

use crate::cli::parser::{Cli, Commands, DevelCommands, CaveCommands};
use crate::logging::init::init_logging;
use crate::models::config::Config;
use clap::Parser;

fn main() {
    let cli = Cli::parse();
    let config = Config::new();

    init_logging(cli.verbose);

    match cli.command {
        Commands::Version => {
            println!("pi version: {}", build::BUILD_VERSION);
            println!("build date: {}", build::BUILD_DATE);
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
            cli::parser::PackageCommands::List { selector } => {
                commands::package::list::run(&config, selector.as_deref());
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
            CaveCommands::Add { arg1, arg2 } => commands::cave::add::run(&config, arg1, arg2),
            CaveCommands::Rem { arg1, arg2 } => commands::cave::rem::run(&config, arg1, arg2),
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
