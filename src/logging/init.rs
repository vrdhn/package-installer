use log::LevelFilter;

pub fn init_logging(quiet: bool, verbose: bool, debug: bool) {
    let log_level = if debug {
        LevelFilter::Trace
    } else if verbose {
        LevelFilter::Debug
    } else if quiet {
        LevelFilter::Error
    } else {
        LevelFilter::Info
    };

    env_logger::Builder::new()
        .filter_level(log_level)
        .format_timestamp(None)
        .format_target(false)
        .init();
}
