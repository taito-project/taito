// Package httputil provides a shared HTTP client with sensible defaults
// for all outbound HTTP requests made by taito.
package httputil

import (
	"net/http"
	"time"
)

// Client is a shared HTTP client with a 30-second timeout.
// All packages making HTTP requests should use this instead of
// http.DefaultClient to prevent indefinite hangs on unresponsive servers.
var Client = &http.Client{
	Timeout: 30 * time.Second,
}
