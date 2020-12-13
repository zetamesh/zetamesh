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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lonng/zetamesh/version"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Client is used to access with the remote gateway
type Client struct {
	gateway string
	key     string
	tls     bool
}

// NewClient returns a new client instance which can be used to interact
// with the gateway.
func NewClient(gateway, key string, tls bool) *Client {
	return &Client{
		gateway: gateway,
		key:     key,
		tls:     tls,
	}
}

// OpenTunnel request the server to open tunnel between the two peers.
// The source and destionation virtual network address need to be provided.
func (c *Client) OpenTunnel(src, dst string) error {
	req := OpenTunnelRequest{
		Version:     version.NewVersion().String(),
		Source:      src,
		Destination: dst,
	}
	res := OpenTunnelResponse{}
	err := c.post(URIOpenTunnel, req, &res)
	if err != nil {
		return errors.WithMessage(err, "open tunnel failed")
	}

	return nil
}

func (c *Client) do(method, api string, reader io.Reader, res interface{}) error {
	var prefix string
	if c.tls {
		prefix = fmt.Sprintf("https://%s", c.gateway)
	} else {
		prefix = fmt.Sprintf("http://%s", c.gateway)
	}
	url := prefix + api
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		return errors.WithStack(err)
	}

	// Set the request headers
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	zap.L().Info("request", zap.Reflect("req", request.Method))

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.WithStack(err)
	}
	zap.L().Info("request", zap.Reflect("req", request.Method))

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	result := &Result{Data: res}
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return errors.WithMessagef(err, "invalid json response when request %s", api)
	}

	if !isSuccess(result.Code) {
		return ErrorWithCode(result.Code, errors.New(result.Error))
	}

	return nil
}

//nolint:unused
func (c *Client) get(api string, res interface{}) error {
	return c.do(http.MethodGet, api, nil, res)
}

func (c *Client) post(api string, req, res interface{}) error {
	buffer := &bytes.Buffer{}
	err := json.NewEncoder(buffer).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}

	return c.do(http.MethodPost, api, buffer, res)
}
