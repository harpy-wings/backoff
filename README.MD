# backoff

A small, production-oriented Go package for retrying flaky operations with **exponential backoff** and **jitter**. It is context-aware, so sleeps stop as soon as the context is canceled or times out—useful in servers, workers, and serverless runtimes where you should not hold resources after the caller has given up.

## Features

- **Context-aware retries** — Checks `context.Context` before each attempt and between sleeps; a canceled or expired context ends the retry loop immediately.
- **Exponential backoff with cap** — Delay grows as `initialDelay × 2^attempt`, bounded by `maxDelay`.
- **Jitter** — Adds up to 10% random extra delay per wait to spread concurrent retries and reduce thundering-herd effects.
- **Two APIs** — `ExponentialDo` for `error`-only work; `ExponentialDoWithReturn` for generic functions that return a value and an error.
- **Clear errors** — Joins context cancellation with the last function error; preserves the last error when all attempts are exhausted.

## Installation

```bash
go get github.com/harpy-wings/backoff
```

## Quick start

### Retries without a return value

Use `ExponentialDo` when your operation only returns an `error` (HTTP calls, DB writes, message publishing, etc.).

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/harpy-wings/backoff"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := backoff.ExponentialDo(ctx, func(ctx context.Context) error {
		return callExternalAPI(ctx)
	},
		backoff.WithMaxAttempts(4),
	)
	if err != nil {
		fmt.Printf("operation failed: %v\n", err)
	}
}
```

### Retries with a return value

Use `ExponentialDoWithReturn` when you need data back from the operation (RPC reads, cache fetches, etc.). On failure it returns the zero value of `T` and an error.

```go
package main

import (
	"context"
	"fmt"

	"github.com/harpy-wings/backoff"
)

type UserBalance struct {
	Amount string
}

func main() {
	ctx := context.Background()

	balance, err := backoff.ExponentialDoWithReturn[UserBalance](ctx, func(ctx context.Context) (UserBalance, error) {
		return fetchBalance(ctx)
	})
	if err != nil {
		fmt.Printf("fetch failed: %v\n", err)
		return
	}

	fmt.Printf("balance: %s\n", balance.Amount)
}
```

## How it works

For each failed attempt (except the last), the package waits before trying again:

1. **Base delay** — `initialDelay × 2^attemptIndex` (0-based index).
2. **Cap** — If the base delay exceeds `maxDelay`, it is clamped to `maxDelay`.
3. **Jitter** — A random duration in `[0, 10% of the delay]` is added.
4. **Sleep** — Waits on a timer, but returns immediately if `ctx` is done.

The function runs up to `maxAttempts` times. The first attempt runs with no prior sleep.

## Configuration

Defaults are applied when you omit options:

| Option | Default | Description |
|--------|---------|-------------|
| `WithMaxAttempts(n)` | `5` | Maximum number of attempts (including the first). |
| `WithInitialDelay(d)` | `10ms` | Base delay before the first retry. |
| `WithMaxDelay(d)` | `10s` | Upper bound on the computed delay. |

Example with custom timing:

```go
err := backoff.ExponentialDo(ctx, fn,
	backoff.WithMaxAttempts(6),
	backoff.WithInitialDelay(50*time.Millisecond),
	backoff.WithMaxDelay(30*time.Second),
)
```

## Context cancellation

If the context is canceled or its deadline passes:

- **Before an attempt** — The loop stops and returns `ctx.Err()` (no join with the last function error).
- **During a sleep** — Returns `errors.Join(ctx.Err(), lastErr)` so you see both why retries stopped and what the last attempt returned.

After all attempts fail without success, `ExponentialDo` returns a joined error that includes the last function error and a summary that retries were exhausted. `ExponentialDoWithReturn` returns the zero value of `T` and wraps the last error in a failure message.

## License

See [LICENSE](LICENSE).
