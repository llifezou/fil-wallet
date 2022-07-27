package main

import (
	logging "github.com/ipfs/go-log/v2"
	"github.com/llifezou/fil-wallet/wallet"
	"github.com/urfave/cli/v2"
	"os"
)

var log = logging.Logger("fil-wallet")

func main() {
	_ = logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:                 "fil-wallet",
		Usage:                "fil wallet",
		Version:              "v0.1.0",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			wallet.Cmd,
			wallet.ChainCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		os.Exit(1)
		return
	}
}
