package main

import (
	"context"
	"fmt"
	multichain_transaction_syncs "github.com/dapplink-labs/multichain-sync-account"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/dapplink-labs/multichain-sync-account/common/cliapp"
	"github.com/dapplink-labs/multichain-sync-account/common/opio"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	flags2 "github.com/dapplink-labs/multichain-sync-account/flags"
	"github.com/dapplink-labs/multichain-sync-account/services"
)

func runMultichainSync(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("exec wallet sync")
	cfg, err := config.LoadConfig(ctx)
	fmt.Println()
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}
	return multichain_transaction_syncs.NewMultiChainSync(ctx.Context, &cfg, shutdown)
}

func runRpc(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	fmt.Println("running grpc server...")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}
	grpcServerCfg := &services.BusinessMiddleConfig{
		GrpcHostname: cfg.RpcServer.Host,
		GrpcPort:     cfg.RpcServer.Port,
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return nil, err
	}
	return services.NewBusinessMiddleWireServices(db, grpcServerCfg, nil)
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
	return db.ExecuteSQLMigration(cfg.Migrations + "/" + strings.ToLower(cfg.ChainNode.TradingModel))
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
				Name:        "sync",
				Flags:       flags,
				Description: "Run rpc scanner wallet chain node",
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
