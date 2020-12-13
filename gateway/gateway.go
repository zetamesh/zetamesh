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

package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lonng/zetamesh/api"
	"github.com/lonng/zetamesh/constant"
	"github.com/lonng/zetamesh/message"
	"github.com/pingcap/fn"
	"go.uber.org/zap"
)

// Options repsents the CLI arguments of Zetamesh gateway node
type Options struct {
	Host        string
	Port        int
	Concurrency int
	Key         string
	TLSCert     string
	TLSKey      string
}

// setupMiddleware is used to setting up all middlewares, e.g:
// 1. Load all plugins
// 2. Set the error encoder
// 3. set the response encoder
func setupMiddleware() {
	fn.Plugin(func(ctx context.Context, request *http.Request) (context.Context, error) {
		return context.WithValue(ctx, api.KeyRawRequest, request), nil
	})

	// Define a error encoder to unify all error response
	fn.SetErrorEncoder(func(ctx context.Context, err error) interface{} {
		request := ctx.Value(api.KeyRawRequest).(*http.Request)
		zap.L().Error("Handle HTTP API request failed",
			zap.String("api", request.RequestURI),
			zap.String("method", request.Method),
			zap.String("remote", request.RemoteAddr),
			zap.Error(err))

		code := message.StatusCode_ServerInternal
		if e, ok := err.(*api.Error); ok {
			code = e.Code
		}
		return &api.Result{
			Code:  code,
			Error: err.Error(),
		}
	})

	// Define a body response to unify all success response
	fn.SetResponseEncoder(func(ctx context.Context, payload interface{}) interface{} {
		request := ctx.Value(api.KeyRawRequest).(*http.Request)
		zap.L().Debug("Handle HTTP API request success",
			zap.String("api", request.RequestURI),
			zap.String("method", request.Method),
			zap.String("remote", request.RemoteAddr),
			zap.Reflect("data", payload))
		return &api.Result{
			Data: payload,
		}
	})
}

// Serve serves the gateway service
func Serve(opt Options) error {
	setupMiddleware()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: opt.Port})
	if err != nil {
		zap.L().Fatal("Listen UDP port failed", zap.Error(err))
	}
	defer conn.Close()

	zap.L().Info("Listen UDP successfully", zap.Int("port", opt.Port))

	var (
		notifier  = newNotifier()
		server    = api.NewServer(notifier, opt.Key)
		processor = newProcessor(server, notifier)
		buffer    = make([]byte, constant.MaxBufferSize)
	)

	// Serve the notifier service
	go notifier.start(conn, opt.Concurrency)

	// Initialize the HTTP service and register all APIs
	router := mux.NewRouter()
	router.Handle(api.URIOpenTunnel, fn.Wrap(server.OpenTunnel)).Methods(http.MethodPost)

	go func() {
		var err error
		addr := fmt.Sprintf("%s:%d", opt.Host, opt.Port)
		if len(opt.TLSCert) > 0 {
			err = http.ListenAndServeTLS(addr, opt.TLSCert, opt.TLSKey, router)
		} else {
			err = http.ListenAndServe(addr, router)
		}
		if err != nil {
			zap.L().Fatal("Listen HTTP port failed", zap.Error(err))
		}
	}()

	for {
		n, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			zap.L().Error("Read UDP packet failed", zap.Error(err))
			continue
		}

		if n < 1 {
			zap.L().Error("Read invalid ackPeer packet", zap.Stringer("remote", remote))
			continue
		}

		if err := processor.process(remote, buffer[:n]); err != nil {
			zap.L().Error("Process message failed", zap.Error(err))
		}
	}
}
