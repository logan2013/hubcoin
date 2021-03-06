package main

import (
	"os"

	"github.com/dragosroua/hubcoin/cmd/commands"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "hubcoin"
	app.Usage = "hubcoin [command] [args...]"
	app.Version = "0.1.0"
	app.Commands = []cli.Command{
		commands.StartCmd,
		commands.TxCmd,
		commands.QueryCmd,
		commands.KeyCmd,
		commands.VerifyCmd,
		commands.BlockCmd,
		commands.AccountCmd,
	}
	app.Run(os.Args)
}
