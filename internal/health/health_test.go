package health

import (
	"context"
	"testing"
	"time"
)

func TestThresholds(t *testing.T) {
	up := false
	r, e := New(Check{Kind: Process, Interval: time.Millisecond, Timeout: time.Second, SuccessThreshold: 2, FailureThreshold: 2, Process: func() bool { return up }})
	if e != nil {
		t.Fatal(e)
	}
	r.runOnce(context.Background(), nil)
	if r.Result().Healthy {
		t.Fatal("healthy too early")
	}
	up = true
	r.runOnce(context.Background(), nil)
	r.runOnce(context.Background(), nil)
	if !r.Result().Healthy {
		t.Fatal("not healthy")
	}
	up = false
	r.runOnce(context.Background(), nil)
	if !r.Result().Healthy {
		t.Fatal("unhealthy too early")
	}
	r.runOnce(context.Background(), nil)
	if r.Result().Healthy {
		t.Fatal("still healthy")
	}
}
