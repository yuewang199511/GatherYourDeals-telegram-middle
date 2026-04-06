package client

import "time"

func testCBConfig() CBConfig {
	return CBConfig{FailureThreshold: 5, SuccessThreshold: 2, OpenTimeout: 30 * time.Second}
}
