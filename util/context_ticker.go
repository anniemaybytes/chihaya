package util

import (
	"context"
	"time"
)

func ContextTick(ctx context.Context, d time.Duration, onTick func()) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			onTick()
		}
	}
}
