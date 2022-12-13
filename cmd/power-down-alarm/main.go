package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/cg14823/power-down-alarm/mailer"
	"github.com/cg14823/power-down-alarm/pulse"
	"github.com/cg14823/power-down-alarm/store"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	atom := zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()
}

func main() {
	app := &cli.App{
		Name:    "power-down-alarm",
		Usage:   "Writes timeframes in which the system was not on",
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db",
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
			&cli.StringFlag{
				Name:     "to",
				Usage:    "The email location to send the test email to",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "from",
				Usage:    "The email to send the notification from",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "token",
				Usage: "The path to the token",
				Value: "/home/pi/.gmail/token.json",
			},
			&cli.StringFlag{
				Name:  "creds",
				Usage: "The path to the client secret",
				Value: "/home/pi/.gmail/client_secret.json",
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
	dbPath, err := filepath.Abs(ctx.String("db"))
	if err != nil {
		return err
	}

	outageStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		return err
	}

	mailer, err := mailer.NewMailer(
		ctx.String("to"),
		ctx.String("from"),
		ctx.String("creds"),
		ctx.String("token"),
		logger,
	)
	if err != nil {
		logger.Sugar().Warnw("could not setup mailer", "err", err)
	}

	return pulse.StartPulser(
		context.Background(),
		outageStore,
		ctx.Duration("frequency"),
		ctx.Duration("threshold"),
		mailer,
		logger,
	)
}
