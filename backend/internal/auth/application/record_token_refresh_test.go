package application

import (
	"minimart/pkg/metrics"
	"testing"
)

func TestRecordTokenRefresh(t *testing.T) {
	svc := &Service{m: metrics.Noop{}}

	svc.RecordTokenRefresh("success")
	svc.RecordTokenRefresh("error")

	// No metrics verification - uses Noop
}
