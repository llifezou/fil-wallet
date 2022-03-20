package main

import (
	logging "github.com/ipfs/go-log/v2"
	"github.com/llifezou/fil-wallet/cmd"
	"github.com/urfave/cli/v2"
	"os"
)

var log = logging.Logger("fil-wallet")

func main() {
	_ = logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:                 "fil-wallet",
		Usage:                "fil wallet",
		Version:              "v0.0.1",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmd.WalletCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		os.Exit(1)
		return
	}
}
