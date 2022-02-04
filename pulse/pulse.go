package pulse

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cg14823/power-down-alarm/store"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func StartPulser(ctx context.Context, path, csvPath string, frequency, threshold time.Duration) error {
	logger.Sugar().Infow("starting pulser", "path", path, "csv", csvPath)
	pulseFile, created, err := getOrCreatePulseFile(path)
	if err != nil {
		return err
	}

	if !created {
		outage, err := checkOutage(pulseFile, threshold)
		if err != nil {
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
	}

	return pulse(ctx, pulseFile, frequency)
}

func pulse(ctx context.Context, pulseFile *os.File, frequency time.Duration) error {
	logger.Sugar().Infow("starting pulse", "frequency", frequency.String())
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("done")
			return nil
		case <-ticker.C:
			if _, err := pulseFile.Write([]byte{0x80}); err != nil {
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
	pulseFile, err := os.OpenFile(absPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.FileMode(0644))
	return pulseFile, created, err
}

type outage struct {
	start time.Time
	end   time.Time
}

func checkOutage(file *os.File, threshold time.Duration) (*outage, error) {
	logger.Sugar().Infow("checking outage", "threshold", threshold.String())
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	logger.Sugar().Infow("last modified", "mod_time", info.ModTime().Format(time.RFC3339Nano))
	if time.Since(info.ModTime()) >= threshold {
		return &outage{info.ModTime().UTC(), time.Now().UTC()}, nil
	}

	return nil, nil
}
