//! High-performance DynamoDB-compatible key-value store exposed via C FFI.
//!
//! This crate replaces the Go DynamoDB TableStore for the hot path:
//! CreateTable, PutItem, GetItem, DeleteItem, ListTables, Query, Scan.
//!
//! All inputs and outputs are raw JSON bytes. The Rust side handles parsing
//! and serialization using simd-json/serde, which is 2-3x faster than Go's
//! encoding/json for typical DynamoDB payloads.

mod store;
mod types;
mod ffi;

pub use ffi::*;
