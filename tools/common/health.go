package common

import (
	"fmt"
	"net/http"
	"time"
)

// WaitForHealth polls the cloudmock health endpoint until it responds with 200
// or the timeout is reached. It polls every 500ms.
func WaitForHealth(endpoint string, timeout time.Duration) error {
	healthURL := endpoint + "/_cloudmock/health"
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("cloudmock not healthy after %s (endpoint: %s)", timeout, healthURL)
		}

		time.Sleep(500 * time.Millisecond)
	}
}
