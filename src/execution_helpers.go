package main

import (
	"fmt"
	"os"
	"time"
)

// verboseLog prints verbose execution details to stderr
func verboseLog(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
}

// applySleep applies a sleep duration if specified
func applySleep(sleepDuration *string, verbose bool) error {
	if sleepDuration == nil || *sleepDuration == "" {
		return nil
	}

	duration, err := time.ParseDuration(*sleepDuration)
	if err != nil {
		return fmt.Errorf("invalid sleep duration '%s': %w", *sleepDuration, err)
	}

	if verbose {
		verboseLog("Sleeping for %s...", duration)
	}

	time.Sleep(duration)
	return nil
}

// parseTimeoutConfig parses timeout configuration from raw JSON
// Returns: (intervalDuration, customErrorCode, error)
func parseTimeoutConfig(timeoutRaw []byte) (time.Duration, *int, error) {
	if len(timeoutRaw) == 0 {
		return 0, nil, nil
	}

	interval, errorCode, err := ParseTimeout(timeoutRaw)
	if err != nil {
		return 0, nil, err
	}

	if interval == "" {
		return 0, errorCode, nil
	}

	duration, err := time.ParseDuration(interval)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid timeout interval '%s': %w", interval, err)
	}

	return duration, errorCode, nil
}
