use log::LevelFilter;

pub fn init_logging(verbose: bool) {
    let log_level = if verbose {
        LevelFilter::Trace
    } else {
        LevelFilter::Info
    };

    env_logger::Builder::new().filter_level(log_level).init();
}
