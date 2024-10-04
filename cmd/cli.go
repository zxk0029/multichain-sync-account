package main

import (
	"context"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/dapplink-labs/multichain-transaction-syncs/common/cliapp"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/opio"
	"github.com/dapplink-labs/multichain-transaction-syncs/config"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	flags2 "github.com/dapplink-labs/multichain-transaction-syncs/flags"
)

func runMultichainSync(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	return nil, nil
}

func runRpc(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	return nil, nil
}

func runGenerateAddress(ctx *cli.Context) error {
	return nil
}

func runMigrations(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("running migrations...")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			log.Error("fail to close database", "err", err)
		}
	}(db)
	return db.ExecuteSQLMigration(cfg.Migrations)
}

func NewCli(GitCommit string, GitData string) *cli.App {
	flags := flags2.Flags
	return &cli.App{
		Version:              params.VersionWithCommit(GitCommit, GitData),
		Description:          "An exchange wallet scanner services with rpc and rest api server",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "rpc",
				Flags:       flags,
				Description: "Run rpc services",
				Action:      cliapp.LifecycleCmd(runRpc),
			},
			{
				Name:        "generate-address",
				Flags:       flags,
				Description: "Run grenerate adddress tools",
				Action:      runGenerateAddress,
			},
			{
				Name:        "wallet",
				Flags:       flags,
				Description: "Run rpc scanner wallet services",
				Action:      cliapp.LifecycleCmd(runMultichainSync),
			},
			{
				Name:        "migrate",
				Flags:       flags,
				Description: "Run database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "version",
				Description: "Show project version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
