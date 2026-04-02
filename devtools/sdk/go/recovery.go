package cloudmock

import (
	"fmt"
	"net/http"
	"runtime"
)

// Recovery returns an http.Handler middleware that recovers from panics,
// sends error:uncaught events to the devtools server, and re-panics so the
// default behavior (or outer recovery) still applies.
//
// Usage:
//
//	mux := http.NewServeMux()
//	handler := cloudmock.Recovery(cloudmock.WrapHandler(mux))
//	http.ListenAndServe(":8080", handler)
func Recovery(next http.Handler) http.Handler {
	conn := getConn()
	if conn == nil {
		return next
	}
	return &recoveryHandler{next: next, conn: conn}
}

type recoveryHandler struct {
	next http.Handler
	conn *connection
}

func (h *recoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			// Capture the stack trace
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			stack := string(buf[:n])

			name := "panic"
			message := fmt.Sprintf("%v", rec)

			if err, ok := rec.(error); ok {
				name = fmt.Sprintf("%T", err)
				message = err.Error()
			}

			h.conn.send("error:uncaught", map[string]any{
				"name":    name,
				"message": message,
				"stack":   stack,
				"url":     r.URL.String(),
				"method":  r.Method,
			})

			// Return 500 to the client
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	h.next.ServeHTTP(w, r)
}

// RecoverFunc is a standalone panic recovery function for use in goroutines.
// Wrap the body of a goroutine with this to capture panics.
//
// Usage:
//
//	go func() {
//	    defer cloudmock.RecoverFunc()
//	    // ... work that might panic
//	}()
func RecoverFunc() {
	if rec := recover(); rec != nil {
		conn := getConn()
		if conn == nil {
			// Re-panic if SDK not initialized; don't swallow it
			panic(rec)
		}

		buf := make([]byte, 8192)
		n := runtime.Stack(buf, false)
		stack := string(buf[:n])

		name := "panic"
		message := fmt.Sprintf("%v", rec)

		if err, ok := rec.(error); ok {
			name = fmt.Sprintf("%T", err)
			message = err.Error()
		}

		conn.send("error:uncaught", map[string]any{
			"name":    name,
			"message": message,
			"stack":   stack,
		})

		// Re-panic so the caller can still handle it
		panic(rec)
	}
}
