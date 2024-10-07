package main

import (
	"context"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/opio"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

var (
	GitCommit = ""
	GitData   = ""
)

func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	app := NewCli(GitCommit, GitData)
	ctx := opio.WithInterruptBlocker(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error("Application failed")
		os.Exit(1)
	}
}
