use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Repository {
    pub uuid: String,
    pub path: String,
    pub name: String,
}

impl Repository {
    pub fn new(path: String, name: String) -> Self {
        Self {
            uuid: Uuid::new_v4().to_string(),
            path,
            name,
        }
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct RepositoryConfig {
    pub repositories: Vec<Repository>,
}
