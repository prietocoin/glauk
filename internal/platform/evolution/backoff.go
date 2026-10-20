package evolution

import (
	"context"
	"errors"
	"time"
)

const maxBackoff = 5 * time.Second

// backoff calcula retryBase × 2^attempt con jitter ±20 %, respeta Retry-After y nunca supera maxBackoff.
func (c *Client) backoff(attempt int, lastErr error) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	var apiErr *APIError
	if errors.As(lastErr, &apiErr) && apiErr.RetryAfter > 0 {
		delay := time.Duration(apiErr.RetryAfter) * time.Second
		if delay <= 0 || delay > maxBackoff {
			return maxBackoff
		}
		return delay
	}

	base := c.retryBase
	if base <= 0 {
		base = 250 * time.Millisecond
	}
	shift := attempt - 1
	if shift > 4 {
		shift = 4
	}
	delay := base * time.Duration(1<<shift)
	jitter := 0.5
	if c.randFloat != nil {
		jitter = c.randFloat()
	}
	delay = time.Duration(float64(delay) * (0.8 + 0.4*jitter))
	if delay > maxBackoff {
		return maxBackoff
	}
	return delay
}

func waitContext(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
