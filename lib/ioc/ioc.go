package ioc

import (
	"svc-opt/lib/batcher"
	"time"
)

var (
	GET_USER_BY_IDS = "GET_USER_BY_IDS"
	BATCH_MAX_REQ   = 100
	BATCH_MAX_WAIT  = 100 * time.Millisecond
	BatchIOC        batcher.Batcher
)
