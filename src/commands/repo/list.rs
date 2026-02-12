use crate::models::config::Config;
use crate::models::package_entry::PackageList;
use crate::models::repository::Repositories;
use comfy_table::Table;

pub fn run(config: &Config, name: Option<&str>) {
    let repo_config = Repositories::get_all(config);

    let mut table = Table::new();
    table.set_header(vec!["Repo Name", "Type", "Name", "Discover Fn"]);

    for repo in &repo_config.repositories {
        if let Some(target_name) = name {
            if repo.name != target_name {
                continue;
            }
        }

        if let Some(package_list) = PackageList::get_for_repo(config, repo) {
            for pkg in &package_list.packages {
                table.add_row(vec![
                    repo.name.clone(),
                    "Package".to_string(),
                    pkg.name.clone(),
                    pkg.function_name.clone(),
                ]);
            }

            for mgr in &package_list.managers {
                table.add_row(vec![
                    repo.name.clone(),
                    "Manager".to_string(),
                    mgr.name.clone(),
                    mgr.function_name.clone(),
                ]);
            }
        }
    }

    println!("{table}");
}

