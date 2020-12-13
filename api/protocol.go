// Copyright 2020 ZetaMesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import "github.com/lonng/zetamesh/message"

type (
	// Result represent the common part of API response
	Result struct {
		Code  message.StatusCode `json:"code"`
		Error string             `json:"error,omitempty"`
		Data  interface{}        `json:"data,omitempty"`
	}

	// OpenTunnelRequest represents the request when trying to open
	// a tunnel
	OpenTunnelRequest struct {
		Version     string `json:"version"`
		Algorithm   string `json:"algorithm"`
		Nonce       string `json:"nonce"`
		Cipher      string `json:"cipher"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}

	// OpenTunnelResponse represent the response of trying to open
	// a tunnel
	OpenTunnelResponse struct {
		Encrypt string `json:"encrypt"`
	}
)
