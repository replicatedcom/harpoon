package main

import (
	"fmt"
	"os"

	"github.com/replicatedhq/harpoon/dockerreg"
	"github.com/replicatedhq/harpoon/dockerreg/v1"
	"github.com/replicatedhq/harpoon/dockerreg/v2"

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
	app.Usage = "Pull any Docker container.  From anywhere."
	app.Commands = []cli.Command{
		{
			Name:   "pull",
			Usage:  "pull a Docker image",
			Action: handlerPull,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "proxy"},
				cli.BoolFlag{Name: "no-cache"},
				cli.BoolFlag{Name: "no-load"},
				cli.BoolFlag{Name: "force-v1"},
				cli.StringFlag{Name: "token"},
			},
		},
	}

	app.Run(os.Args)
}

func handlerPull(c *cli.Context) error {
	fmt.Printf("Pulling image %q\n", c.Args()[0])

	dockerRemote, err := dockerreg.ParseDockerURI(c.Args()[0])
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	if c.Bool("v1") {
		if err := v1.PullImage(dockerRemote, c.String("proxy"), c.Bool("no-cache")); err != nil {
			fmt.Println(err.Error())
			return err
		}
	} else {
		// TODO, this should fall back to v1 if needed?  Or is v1 deprecated enough to justify leaving this to fail?
		if err := v2.PullImage(dockerRemote, c.String("proxy"), c.Bool("no-cache"), c.String("token")); err != nil {
			fmt.Println(err.Error())
			return err
		}
	}

	return nil
}
