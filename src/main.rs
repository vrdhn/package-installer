mod build;
mod cli;
mod commands;
mod logging;
mod models;
mod services;
mod starlark;

use crate::cli::parser::{Cli, Commands, DevelCommands};
use crate::logging::init::init_logging;
use clap::Parser;

fn main() {
    let cli = Cli::parse();

    init_logging(cli.verbose);

    match cli.command {
        Commands::Version => {
            println!("pi version: {}", build::BUILD_VERSION);
            println!("build date: {}", build::BUILD_DATE);
        }
        Commands::Repo { command } => match command {
            cli::parser::RepoCommands::Add { path } => {
                commands::repo::add::run(&path);
            }
            cli::parser::RepoCommands::Sync { name } => {
                commands::repo::sync::run(name.as_deref());
            }
            cli::parser::RepoCommands::List { name } => {
                commands::repo::list::run(name.as_deref());
            }
        },
        Commands::Package { command } => match command {
            cli::parser::PackageCommands::Sync { selector } => {
                commands::package::sync::run(selector.as_deref());
            }
            cli::parser::PackageCommands::List { selector } => {
                commands::package::list::run(selector.as_deref());
            }
        },
        Commands::Devel { command } => match command {
            DevelCommands::Test { filename, pkg } => {
                commands::devel::test::run(&filename, pkg.as_deref());
            }
        },
    }
}
