use std::ffi::{c_char, CStr};
use std::ptr;
use std::slice;

use crate::store::DdbStore;

/// Opaque handle to a DdbStore.
pub type DdbStoreHandle = *mut DdbStore;

/// Result of a DynamoDB operation: status code + response bytes.
#[repr(C)]
pub struct DdbResult {
    pub status: u16,
    pub data: *mut u8,
    pub len: usize,
}

/// Create a new DdbStore. Caller must eventually call ddb_store_free.
#[unsafe(no_mangle)]
pub extern "C" fn ddb_store_new(
    account_id: *const c_char,
    region: *const c_char,
) -> DdbStoreHandle {
    let account = unsafe { CStr::from_ptr(account_id) }
        .to_str()
        .unwrap_or("000000000000")
        .to_string();
    let reg = unsafe { CStr::from_ptr(region) }
        .to_str()
        .unwrap_or("us-east-1")
        .to_string();

    Box::into_raw(Box::new(DdbStore::new(account, reg)))
}

/// Free a DdbStore.
#[unsafe(no_mangle)]
pub extern "C" fn ddb_store_free(handle: DdbStoreHandle) {
    if !handle.is_null() {
        unsafe {
            drop(Box::from_raw(handle));
        }
    }
}

/// Handle a DynamoDB request.
///
/// action: null-terminated C string (e.g., "PutItem")
/// body: pointer to JSON bytes
/// body_len: length of body
///
/// Returns a DdbResult. If status == 0, the action is not handled by Rust
/// and the caller should fall back to Go. The caller must free result.data
/// with ddb_result_free.
#[unsafe(no_mangle)]
pub extern "C" fn ddb_handle(
    handle: DdbStoreHandle,
    action: *const c_char,
    body: *const u8,
    body_len: usize,
) -> DdbResult {
    if handle.is_null() || action.is_null() {
        return DdbResult {
            status: 0,
            data: ptr::null_mut(),
            len: 0,
        };
    }

    let store = unsafe { &*handle };
    let action_str = unsafe { CStr::from_ptr(action) }
        .to_str()
        .unwrap_or("");
    let body_slice = if body.is_null() || body_len == 0 {
        &[]
    } else {
        unsafe { slice::from_raw_parts(body, body_len) }
    };

    let (status, response) = store.handle(action_str, body_slice);

    if status == 0 {
        // Not handled — fall back to Go.
        return DdbResult {
            status: 0,
            data: ptr::null_mut(),
            len: 0,
        };
    }

    let len = response.len();
    let data = if len > 0 {
        let mut boxed = response.into_boxed_slice();
        let ptr = boxed.as_mut_ptr();
        std::mem::forget(boxed);
        ptr
    } else {
        ptr::null_mut()
    };

    DdbResult { status, data, len }
}

/// Free the data pointer returned by ddb_handle.
#[unsafe(no_mangle)]
pub extern "C" fn ddb_result_free(data: *mut u8, len: usize) {
    if !data.is_null() && len > 0 {
        unsafe {
            drop(Vec::from_raw_parts(data, len, len));
        }
    }
}
