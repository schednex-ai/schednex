package backoff_config

import "time"

const (
	MAX_TIME     = time.Duration(time.Second * 60 * 10)
	MAX_INTERVAL = time.Duration(time.Second * 60)
)
