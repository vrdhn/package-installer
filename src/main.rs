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
        Commands::Devel { command } => match command {
            DevelCommands::Test { filename, pkg } => {
                commands::devel::test::run(&filename, pkg.as_deref());
            }
        },
    }
}
