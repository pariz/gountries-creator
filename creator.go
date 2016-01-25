package main

import (
	"os"

	"github.com/codegangsta/cli"
)

func main() {

	app := cli.NewApp()

	app.Name = "Gountries Creator"

	// Create files from data
	//

	app.Commands = []cli.Command{
		{
			Name:   "create",
			Usage:  "Generates all data based on the local files",
			Action: createFiles,
		},
		{
			Name:   "import",
			Usage:  "Imports data from various data sources",
			Action: importFiles,
		},
	}

	app.Run(os.Args)

}
