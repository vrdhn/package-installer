use redb::{Database, TableDefinition};
use std::path::Path;
use anyhow::Result;
use std::time::{SystemTime, UNIX_EPOCH};

pub const CACHE_TABLE: TableDefinition<&str, (&str, u64)> = TableDefinition::new("cache_metadata");
pub const JOURNAL_TABLE: TableDefinition<&str, (&str, u64)> = TableDefinition::new("journal");
pub const INSTALL_TABLE: TableDefinition<&str, u64> = TableDefinition::new("installed_packages");

#[derive(Debug)]
pub struct Db {
    database: Database,
}

impl Db {
    pub fn open(path: &Path) -> Result<Self> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        let database = Database::builder()
            .create(path)?;
            
        Ok(Self { database })
    }

    pub fn set_cache_metadata(&self, url: &str, file_path: &str, expires: u64) -> Result<()> {
        let write_txn = self.database.begin_write()?;
        {
            let mut table = write_txn.open_table(CACHE_TABLE)?;
            table.insert(url, (file_path, expires))?;
        }
        write_txn.commit()?;
        Ok(())
    }

    pub fn get_cache_metadata(&self, url: &str) -> Result<Option<(String, u64)>> {
        let read_txn = self.database.begin_read()?;
        let table = read_txn.open_table(CACHE_TABLE)?;
        let result = table.get(url)?;
        Ok(result.map(|v| {
            let (path, exp) = v.value();
            (path.to_string(), exp)
        }))
    }

    pub fn log_operation(&self, path: &str, operation: &str) -> Result<()> {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        let write_txn = self.database.begin_write()?;
        {
            let mut table = write_txn.open_table(JOURNAL_TABLE)?;
            table.insert(path, (operation, now))?;
        }
        write_txn.commit()?;
        Ok(())
    }

    pub fn get_operation(&self, path: &str) -> Result<Option<(String, u64)>> {
        let read_txn = self.database.begin_read()?;
        let table = read_txn.open_table(JOURNAL_TABLE)?;
        let result = table.get(path)?;
        Ok(result.map(|v| {
            let (op, ts) = v.value();
            (op.to_string(), ts)
        }))
    }

    pub fn is_operation_done(&self, path: &str, operation: &str) -> bool {
        match self.get_operation(path) {
            Ok(Some((op, _))) => op == operation,
            _ => false,
        }
    }

    pub fn mark_installed(&self, cave: &str, variant: Option<&str>, package_id: &str) -> Result<()> {
        let variant = variant.unwrap_or("default");
        let key = format!("{}:{}:{}", cave, variant, package_id);
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        let write_txn = self.database.begin_write()?;
        {
            let mut table = write_txn.open_table(INSTALL_TABLE)?;
            table.insert(key.as_str(), now)?;
        }
        write_txn.commit()?;
        Ok(())
    }

    pub fn is_installed(&self, cave: &str, variant: Option<&str>, package_id: &str) -> Result<bool> {
        let variant = variant.unwrap_or("default");
        let key = format!("{}:{}:{}", cave, variant, package_id);
        let read_txn = self.database.begin_read()?;
        let table = read_txn.open_table(INSTALL_TABLE)?;
        Ok(table.get(key.as_str())?.is_some())
    }
}
