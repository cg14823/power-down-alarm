package main

import (
	"os"
	"time"

	"github.com/cg14823/power-down-alarm/mailer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "mailer-test",
		Usage:   "Send test power outage email",
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "to",
				Usage:    "The email location to send the test email to",
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
		Action: sendEmail,
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func createLogger() *zap.Logger {
	atom := zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()
	return logger
}

func sendEmail(ctx *cli.Context) error {
	logger := createLogger()
	defer logger.Sync()

	logger.Sugar().Infow("Sending test email", "to", ctx.String("to"))
	mailer, err := mailer.NewMailer(
		ctx.String("to"),
		ctx.String("to"),
		ctx.String("creds"),
		ctx.String("token"),
		logger,
	)
	if err != nil {
		return err
	}

	done := mailer.AysncOutageNotification(time.Now().Add(-50*time.Minute), time.Now())
	<-done
	return nil
}
