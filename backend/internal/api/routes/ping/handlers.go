// Package ping contains handlers for pinging the server
package ping

import "net/http"

// HandlePing godoc
//
//	@Summary	Ping endpoint.
//	@Tags		Ping
//
//	@Success	200
//	@Router		/api/ping [GET]
func HandlePing(w http.ResponseWriter, r *http.Request) {}
