package backoff

import "time"

type Option func(*config)

// WithMaxAttempts set the maximum number of attempts
func WithMaxAttempts(maxAttempts int) Option {
	return func(c *config) {
		c.MaxAttempts = maxAttempts
	}
}

// WithInitialDelay set the initial delay
func WithInitialDelay(initialDelay time.Duration) Option {
	return func(c *config) {
		c.InitialDelay = initialDelay
	}
}

// WithMaxDelay set the maximum delay
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(c *config) {
		c.MaxDelay = maxDelay
	}
}
