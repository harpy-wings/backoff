package backoff

import "time"

type config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}
