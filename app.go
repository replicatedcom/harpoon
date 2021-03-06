package main

import (
	"os"

	"github.com/replicatedcom/harpoon/importer"
	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/remote"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "lang",
			Value: "english",
			Usage: "language for the greeting",
		},
	}

	app.Name = "harpoon"
	app.Usage = "Pull any Docker image.  From anywhere."
	app.Commands = []cli.Command{
		{
			Name:   "pull",
			Usage:  "pull a Docker image",
			Action: handlerPull,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "proxy"},
				cli.BoolFlag{Name: "no-load"},
				cli.BoolFlag{Name: "force-v1"},
				cli.StringFlag{Name: "token"},
			},
		},
	}

	app.Run(os.Args)
}

func handlerPull(c *cli.Context) error {
	log.Debugf("Pulling image %q", c.Args()[0])

	dockerRemote, err := remote.ParseDockerURI(c.Args()[0])
	if err != nil {
		log.Debugf("%v", err)
		return err
	}

	// TODO: Tell it to use force v1 if needed
	if err := importer.ImportFromRemote(dockerRemote); err != nil {
		log.Debugf("%v", err)
		return err
	}

	return nil
}
