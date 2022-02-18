package pulse

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cg14823/power-down-alarm/store"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const filePerm = 0644

var logger *zap.Logger

type outage struct {
	start time.Time
	end   time.Time
}

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

func StartPulser(ctx context.Context, path, csvPath string, frequency, threshold time.Duration) error {
	logger.Sugar().Infow("starting pulser", "path", path, "csv", csvPath)
	outage, err := checkOutage(path, threshold)
	if err != nil {
		logger.Sugar().Errorw("could not check outage", "err", err)
		return err
	}

	if outage != nil {
		logger.Sugar().Infow("recording outage", "start", outage.start.Format(time.RFC3339),
			"end", outage.end.Format(time.RFC3339))
		store, err := store.NewCSVOutageStore(csvPath)
		if err != nil {
			return err
		}

		err = store.RecordOutage(outage.start, outage.end)
		if err != nil {
			logger.Sugar().Errorw("could not record outage", "err", err.Error())
			return fmt.Errorf("could not record outage: %w", err)
		}

		store.Close()
	}

	pulseFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		logger.Sugar().Errorw("could not create pulse file", "err", err)
		return err
	}

	return pulse(ctx, pulseFile, frequency)
}

func checkOutage(filePath string, threshold time.Duration) (*outage, error) {
	pulseFile, err := os.OpenFile(filePath, os.O_RDONLY, filePerm)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	defer func() {
		pulseFile.Close()
		os.Remove(filePath)
	}()

	timeBuf := make([]byte, 10)
	_, err = pulseFile.Read(timeBuf)
	if err != nil {
		return nil, err
	}

	lastMod, err := binary.ReadVarint(bytes.NewBuffer(timeBuf))
	if err != nil {
		return nil, err
	}

	lastModTime := time.Unix(lastMod, 0).UTC()
	logger.Sugar().Infow("last modified", "mod_time", time.Unix(lastMod, 0).Format(time.RFC3339))
	if time.Now().UTC().Sub(lastModTime) >= threshold {
		return &outage{lastModTime, time.Now().UTC()}, nil
	}

	return nil, nil
}

func pulse(ctx context.Context, pulseFile *os.File, frequency time.Duration) error {
	logger.Sugar().Infow("starting pulse", "frequency", frequency.String())
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	buf := make([]byte, 10)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("done")
			return nil
		case <-ticker.C:
			binary.PutVarint(buf, time.Now().UTC().Unix())
			if _, err := pulseFile.WriteAt(buf, 0); err != nil {
				logger.Sugar().Errorw("could not right pulse", "err", err.Error())
			}

			pulseFile.Sync()
		}
	}
}

func getOrCreatePulseFile(path string) (*os.File, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, false, err
	}

	_, err = os.Stat(absPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}

	created := err != nil
	pulseFile, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	return pulseFile, created, err
}
