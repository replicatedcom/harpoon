package main

import (
	"fmt"
	"os"

	"github.com/replicatedcom/harpoon/dockerreg"

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

	// TODO: Tell it to use force v1 if needed
	if err := dockerreg.ImportFromRemote(dockerRemote, c.String("proxy")); err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
