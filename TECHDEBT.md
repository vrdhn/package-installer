# Tech Debt & Architectural Critique

## Overview
The current architecture relies on `rayon` for parallelism and `ureq` for HTTP requests. While this avoids the complexity of `tokio`'s async runtime, it introduces significant bottlenecks when mixing CPU-bound tasks (Starlark evaluation, JSON parsing) with I/O-bound tasks (network downloads).

## Critical Issues (High Priority)

### 1. Blocking I/O Saturation in Rayon
**Location:** `src/services/downloader.rs`, `src/commands/cave/resolve.rs`
- **Issue:** The `Downloader` uses `ureq` which is synchronous. When `cave resolve` runs via `par_iter`, it schedules these blocking network calls onto Rayon's worker threads. Rayon is optimized for CPU-bound tasks and defaults to a thread count matching physical cores. Blocking these threads with network I/O starves the pool, effectively serializing execution once all cores are waiting on packets.
- **Fix (No-Tokio):**
    - Create a separate, larger `rayon::ThreadPool` specifically for I/O operations (e.g., 50+ threads) to allow high concurrency despite blocking calls.
    - Alternatively, use `std::thread::spawn` for downloads, though managing limits becomes harder.

### 2. No HTTP Connection Pooling
**Location:** `src/services/downloader.rs`
- **Issue:** A new `ureq::Agent` is created inside the `download` function for every request. This prevents TCP connection reuse (Keep-Alive), resulting in unnecessary DNS queries and TCP/TLS handshakes for every file fetched.
- **Fix:** Lift the `ureq::Agent` into the shared `State` struct (inside `Config`) so it can be reused across requests.

### 3. Unbounded Memory Usage in Downloader
**Location:** `src/services/downloader.rs`
- **Issue:** The downloader reads the entire HTTP response into a `Vec<u8>` and then converts it to a `String`. For large files (even accidentally large ones), this will cause OOM crashes.
- **Fix:** Implement a size limit for downloads intended to be parsed as text/scripts. For binary artifacts, stream directly to disk.

## Performance Optimizations (Medium Priority)

### 4. Starlark Environment Re-initialization
**Location:** `src/starlark/runtime/mod.rs`
- **Issue:** `evaluate_file` and `execute_function` call `create_globals()` every time. This reconstructs the Starlark standard library and API bindings for every single function call or file evaluation.
- **Fix:** Initialize `Globals` once, store it in `State`, and clone/reference it for each execution.

### 5. Repetitive Serialization Logic
**Location:** `src/models/package_entry.rs`, `src/models/version_entry.rs`
- **Issue:** `load` and `save` methods contain duplicated logic for path handling and JSON serialization/deserialization errors.
- **Fix:** Extract a generic `JsonStore` trait or helper functions to handle file I/O and Serde operations uniformly.

## Code Quality & Maintenance (Low Priority)

### 6. Hardcoded Magic Strings
**Location:** `src/commands/package/list.rs`, `src/commands/package/info.rs`, `src/commands/package/resolve.rs`
- **Issue:** Strings like `"latest"`, `"stable"`, `"testing"` are hardcoded in multiple places.
- **Fix:** Define these as constants in a common module (e.g., `src/models/types.rs`).

### 7. Fragile Test Paths
**Location:** `src/starlark/runtime/mod.rs`
- **Issue:** Tests use `/tmp/pi-test` directly. This works on Linux but is not cross-platform and can cause conflicts if tests run in parallel or on shared machines.
- **Fix:** Consistently use the `tempfile` crate for all test directory creation.cn
