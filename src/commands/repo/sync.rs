use crate::models::repository::Repositories;
use crate::commands::repo::list;
use crate::models::config::Config;
use rayon::prelude::*;

pub fn run(config: &Config, name: Option<&str>) {
    sync_all(config, name);
    if log::log_enabled!(log::Level::Info) {
        list::run(config, name);
    }
}

pub fn sync_all(config: &Config, name: Option<&str>) {
    let repo_config = Repositories::get_all(config);

    repo_config.repositories.par_iter().for_each(|repo| {
        if let Some(target_name) = name {
            if repo.name != target_name {
                return;
            }
        }

        if let Err(e) = crate::services::sync::sync_repo(config, repo) {
            log::error!("[{}] sync failed: {}", repo.name, e);
        }
    });
}
