package pulse

import (
	"context"
	"fmt"
	"time"

	"github.com/cg14823/power-down-alarm/mailer"
	"github.com/cg14823/power-down-alarm/store"

	"go.uber.org/zap"
)

const filePerm = 0644

var logger *zap.Logger

type outage struct {
	start time.Time
	end   time.Time
}

type pulser struct {
	store     store.OutageStore
	frequency time.Duration
	threshold time.Duration
	mailer    *mailer.Mailer
}

func StartPulser(ctx context.Context, store store.OutageStore, frequency, threshold time.Duration,
	m *mailer.Mailer, l *zap.Logger) error {
	logger = l
	pulser := &pulser{store, frequency, threshold, m}
	return pulser.start(ctx)
}

func (p *pulser) start(ctx context.Context) error {
	logger.Sugar().Infow("starting pulser", "threshold", p.threshold.String(), "frequency", p.frequency.String())
	outage, err := p.checkOutage()
	if err != nil {
		logger.Sugar().Errorw("could not check outage", "err", err)
		return err
	}

	if outage != nil {
		logger.Sugar().Infow("recording outage", "start", outage.start.Format(time.RFC3339),
			"end", outage.end.Format(time.RFC3339))
		if p.mailer != nil {
			p.mailer.AysncOutageNotification(outage.start, outage.end)
		}

		err := p.store.RecordOutage(outage.start, outage.end)
		if err != nil {
			logger.Sugar().Errorw("could not record outage", "err", err)
			return err
		}
	}

	return p.pulse(ctx)
}

func (p *pulser) checkOutage() (*outage, error) {
	last, err := p.store.GetLastPulse()
	if err != nil {
		return nil, err
	}

	if last.IsZero() {
		return nil, nil
	}

	logger.Sugar().Infow("last modified", "mod_time", last.Format(time.RFC3339))
	if last.After(time.Now().UTC()) {
		logger.Sugar().Warn("last is larger than current time", "last", last.UTC(), "now", time.Now().UTC())
		return nil, nil
	}

	if time.Now().UTC().Sub(last.UTC()) >= p.threshold {
		return &outage{last, time.Now().UTC()}, nil
	}

	return nil, nil
}

func (p *pulser) pulse(ctx context.Context) error {
	logger.Sugar().Infow("starting pulse", "frequency", p.frequency.String())
	ticker := time.NewTicker(p.frequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("done")
			return nil
		case <-ticker.C:
			if err := p.store.Pulse(); err != nil {
				logger.Sugar().Warnw("could not pulse", "err", err)
			}
		}
	}
}
