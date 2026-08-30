package checker

import (
	"context"
	"net/http"
	"pulse/internal/monitor"
	"time"
)

type Result struct {
	Success        bool
	StatusCode     *int
	ResponseTimeMs int
	Error          *string
}

func Check(ctx context.Context, m *monitor.Monitor) Result {
	// Client with timeout so a slow endpoint doesn't block the worker
	client := &http.Client{
		Timeout: time.Duration(m.TimeoutSeconds) * time.Second,
	}

	// Build the request
	req, err := http.NewRequestWithContext(ctx, m.Method, m.URL, nil)
	if err != nil {
		msg := "failed to build request: " + err.Error()
		return Result{Success: false, StatusCode: nil, Error: &msg}
	}

	// Time the request
	start := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(start)

	// Network error or timeout -> failure with not status code
	if err != nil {
		msg := err.Error()
		return Result{
			Success:        false,
			StatusCode:     nil,
			ResponseTimeMs: int(responseTime.Milliseconds()),
			Error:          &msg,
		}
	}
	defer resp.Body.Close()

	// Compare the actual status with the expected one
	success := resp.StatusCode == m.ExpectedStatus
	statusCode := resp.StatusCode

	return Result{
		Success:        success,
		StatusCode:     &statusCode,
		ResponseTimeMs: int(responseTime.Milliseconds()),
		Error:          nil,
	}
}
