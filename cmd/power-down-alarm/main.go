package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/cg14823/power-down-alarm/pulse"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "power-down-alarm",
		Usage:   "Writes timeframes in which the system was not on",
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "pulse-file",
				Usage:    "The file to use for the hearbeat",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "output-csv-file",
				Usage:    "The file to use to store the power cuts",
				Required: true,
			},
			&cli.DurationFlag{
				Name:     "threshold",
				Usage:    "The duration without a pulse to which mark as an outage",
				Required: true,
				Value:    10 * time.Minute,
			},
			&cli.DurationFlag{
				Name:     "frequency",
				Usage:    "The heartbeat frequency",
				Required: true,
				Value:    1 * time.Minute,
			},
		},
		Action: startMonitor,
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func startMonitor(ctx *cli.Context) error {
	pulsePath, err := filepath.Abs(ctx.String("pulse-file"))
	if err != nil {
		return err
	}

	csvPath, err := filepath.Abs(ctx.String("output-csv-file"))
	if err != nil {
		return err
	}

	return pulse.StartPulser(context.Background(), pulsePath, csvPath,
		ctx.Duration("frequency"), ctx.Duration("threshold"))
}
