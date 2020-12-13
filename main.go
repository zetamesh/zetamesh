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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/lonng/zetamesh/gateway"
	"github.com/lonng/zetamesh/node"
	"github.com/lonng/zetamesh/version"
	"github.com/spf13/cobra"
)

func main() {
	if err := execute(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
}

func execute() error {
	rand.Seed(time.Now().UnixNano())

	var (
		logLevel string
	)

	rootCmd := &cobra.Command{
		Use:     "zetamesh",
		Short:   "ZetaMesh is use to establish the link between peers behind NAT",
		Version: version.NewVersion().String(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			var level zapcore.LevelEnabler
			switch strings.ToLower(logLevel) {
			case "error":
				level = zapcore.ErrorLevel
			case "warn":
				level = zapcore.WarnLevel
			case "debug":
				level = zapcore.DebugLevel
			default:
				level = zapcore.InfoLevel
			}
			// Initialize the logger
			config := zap.NewDevelopmentEncoderConfig()
			encoder := zapcore.NewConsoleEncoder(config)
			logger := zap.New(zapcore.NewCore(encoder, os.Stdout, level))
			zap.ReplaceGlobals(logger)
		},
	}

	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Specify the log level(error/warn/info/debug)")

	rootCmd.AddCommand(newGatewayCmd())
	rootCmd.AddCommand(newJoinCmd())

	return rootCmd.Execute()
}

func newGatewayCmd() *cobra.Command {
	var opt gateway.Options

	gatewayCmd := &cobra.Command{
		Use:     "gateway",
		Short:   "Startup a zetamesh gateway server",
		Version: version.NewVersion().String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return gateway.Serve(opt)
		},
	}

	gatewayCmd.Flags().StringVar(&opt.Host, "host", "0.0.0.0", "The serve host of gateway server")
	gatewayCmd.Flags().IntVar(&opt.Port, "port", 2823, "The serve port of gateway server")
	gatewayCmd.Flags().IntVarP(&opt.Concurrency, "concurrency", "c", 128, "The concurrency of sync peer information")
	gatewayCmd.Flags().StringVar(&opt.Key, "key", "", "The key of the gateway, which is used to validate the peers")
	gatewayCmd.Flags().StringVar(&opt.TLSCert, "tls-cert", "", "The tls cert path")
	gatewayCmd.Flags().StringVar(&opt.TLSKey, "tls-key", "", "The tls key path")

	return gatewayCmd
}

func newJoinCmd() *cobra.Command {
	var opt node.Options

	joinCmd := &cobra.Command{
		Use:          "join",
		Short:        "Join to a zetamesh and initialize the local network",
		Version:      version.NewVersion().String(),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: support DHCP
			if opt.Address == "" {
				return cmd.Help()
			}

			node := node.New(opt)
			defer node.Stop()

			return node.Serve()
		},
	}

	joinCmd.Flags().StringVarP(&opt.Gateway, "gateway", "g", "127.0.0.1:2823", "The gateway server address")
	joinCmd.Flags().StringVarP(&opt.Key, "key", "k", "", "The key to connect to the gateway")
	joinCmd.Flags().StringVarP(&opt.Address, "address", "a", "", "(Required)The address of local node")
	joinCmd.Flags().BoolVar(&opt.TLS, "tls", false, "Enable the TLS")

	return joinCmd
}
