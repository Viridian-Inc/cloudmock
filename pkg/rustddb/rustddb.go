// Package rustddb provides Go bindings to the Rust DynamoDB store via CGO.
//
// The Rust store handles CreateTable, PutItem, GetItem, DeleteItem, ListTables
// with simd-json parsing and lock-free DashMap storage. For unsupported actions
// (Query, Scan, BatchWrite, etc.), it returns status=0 and the caller falls
// back to the Go implementation.
package rustddb

/*
#cgo LDFLAGS: -L${SRCDIR}/../../rust/ddb-store/target/release -lddb_store -lm -ldl -framework Security -framework SystemConfiguration
#include <stdint.h>
#include <stdlib.h>

typedef void* DdbStoreHandle;

typedef struct {
    uint16_t status;
    uint8_t* data;
    size_t   len;
} DdbResult;

extern DdbStoreHandle ddb_store_new(const char* account_id, const char* region);
extern void           ddb_store_free(DdbStoreHandle handle);
extern DdbResult      ddb_handle(DdbStoreHandle handle, const char* action, const uint8_t* body, size_t body_len);
extern void           ddb_result_free(uint8_t* data, size_t len);
*/
import "C"

import (
	"unsafe"
)

// Store wraps the Rust DdbStore.
type Store struct {
	handle C.DdbStoreHandle
}

// New creates a Rust-backed DynamoDB store.
func New(accountID, region string) *Store {
	cAccount := C.CString(accountID)
	cRegion := C.CString(region)
	defer C.free(unsafe.Pointer(cAccount))
	defer C.free(unsafe.Pointer(cRegion))

	return &Store{
		handle: C.ddb_store_new(cAccount, cRegion),
	}
}

// Close frees the Rust store.
func (s *Store) Close() {
	if s.handle != nil {
		C.ddb_store_free(s.handle)
		s.handle = nil
	}
}

// Result of a DynamoDB operation.
type Result struct {
	Status int    // HTTP status code. 0 = not handled (fall back to Go).
	Body   []byte // JSON response bytes.
}

// Handle processes a DynamoDB action with the given JSON body.
// Returns status=0 if the action is not handled by the Rust store.
func (s *Store) Handle(action string, body []byte) Result {
	cAction := C.CString(action)
	defer C.free(unsafe.Pointer(cAction))

	var bodyPtr *C.uint8_t
	var bodyLen C.size_t
	if len(body) > 0 {
		bodyPtr = (*C.uint8_t)(unsafe.Pointer(&body[0]))
		bodyLen = C.size_t(len(body))
	}

	r := C.ddb_handle(s.handle, cAction, bodyPtr, bodyLen)

	if r.status == 0 {
		return Result{Status: 0}
	}

	// Copy the Rust-allocated bytes into Go-managed memory, then free.
	var goBytes []byte
	if r.data != nil && r.len > 0 {
		goBytes = C.GoBytes(unsafe.Pointer(r.data), C.int(r.len))
		C.ddb_result_free(r.data, r.len)
	}

	return Result{
		Status: int(r.status),
		Body:   goBytes,
	}
}
