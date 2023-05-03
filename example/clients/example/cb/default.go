package cb

import (
	"time"
)

const defaultInterval = time.Duration(0) * time.Second
const defaultTimeout = time.Duration(10) * time.Second

func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures >= 5
}

func defaultIsSuccessful(err error) bool {
	return err == nil
}
