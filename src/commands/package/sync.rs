use crate::models::repository::Repositories;
use crate::models::package_entry::PackageList;
use crate::commands::package::list;
use crate::models::config::Config;
use crate::models::selector::PackageSelector;
use rayon::prelude::*;

pub fn run(config: &Config, selector_str: Option<&str>) {
    let selector = selector_str.and_then(PackageSelector::parse);
    sync_all(config, selector);
    if log::log_enabled!(log::Level::Info) {
        list::run(config, selector_str);
    }
}

pub fn sync_all(config: &Config, selector: Option<PackageSelector>) {
    let repo_config = Repositories::get_all(config);

    repo_config.repositories.par_iter().for_each(|repo| {
        // If recipe is specified, it must match repo name exactly
        if let Some(ref s) = selector {
            if let Some(ref r_name) = s.recipe {
                if repo.name != *r_name {
                    return;
                }
            }
        }

        if let Some(pkg_list) = PackageList::get_for_repo(config, repo) {
            pkg_list.packages.par_iter().for_each(|pkg| {
                // Match package name exactly
                if let Some(ref s) = selector {
                    if !s.package.is_empty() && s.package != "*" {
                        if pkg.name != s.package {
                            return;
                        }
                    }
                }

                crate::services::sync::sync_package(config, repo, pkg);
            });

            if let Some(ref s) = selector {
                if let Some(ref prefix) = s.prefix {
                    if let Some(mgr) = pkg_list.manager_map.get(prefix) {
                        crate::services::sync::sync_manager_package(
                            config,
                            repo,
                            mgr,
                            prefix,
                            &s.package,
                        );
                    }
                }
            }
        }
    });
}
