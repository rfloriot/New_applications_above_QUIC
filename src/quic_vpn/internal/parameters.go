package internal

import "time"

const (
	readBufSize            = 20000			 // size of the read interface buffer
	inactivityTimeout      = 1 * time.Second // time before closing a flow
	sendQueueMaxSize       = 10000             // max size of send queue
	sendQueueMarkThreshold = 5               // size before starting marking queue
	inactivePollTime  = 500*time.Millisecond
	txMeasurementRefreshTime = 1*time.Second
	qosRefreshTime = 500*time.Millisecond
)
