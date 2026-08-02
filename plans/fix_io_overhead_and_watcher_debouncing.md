# Plan: Fix I/O Overhead and Watcher Debouncing

## Issue Description
1. **Parsing I/O Overhead:** In `analysis/parser.go` (`findFile`), the code enforces case-sensitivity by calling `os.ReadDir(dir)` to read the entire directory contents for every imported file. If a file imports 10 modules from a directory containing 1,000 files, `ReadDir` is called 10 times, causing catastrophic disk I/O thrashing.
2. **Inefficient Debouncing:** The `fsnotify` watcher in `filesystem/watcher.go` spawns a new goroutine using `time.AfterFunc` for every single file event to handle debouncing. During a large git checkout, this floods the system with thousands of short-lived goroutines. Furthermore, manual recursive directory watches risk exceeding OS `inotify` limits.

## Proposed Fix
1. **Optimize Parsing:**
   - Implement a directory cache in the analysis engine to memoize `os.ReadDir` results. When `findFile` needs to check case-sensitivity, it should first consult the cache. The cache can be cleared after a full graph build or incrementally updated.
   - Consider skipping case-sensitivity checks on case-sensitive filesystems (like Linux) or relying on `os.Stat` and handling case-mismatches strictly during resolution.

2. **Optimize Watcher Debouncing:**
   - Refactor `filesystem/watcher.go` to use a single dedicated goroutine with a `time.Ticker` or `time.Timer` loop for debouncing, rather than spawning a goroutine per event.
   - Accumulate incoming events in a map or slice and flush them downstream once the timer expires.

## Execution Steps
1. Add a `sync.Map` or standard map with a mutex to cache directory listings in `analysis/parser.go`.
2. Update `findFile` to use the cache instead of repeatedly calling `os.ReadDir`.
3. Refactor the `startLoop` method in `filesystem/watcher.go` to use a single `select` loop with a `time.Timer` that resets on new events, flushing batched events when the timer fires.
4. Verify parsing speed improvements and ensure branch switching does not cause goroutine spikes.
