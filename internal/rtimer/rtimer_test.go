package rtimer

import (
	"testing"
	"time"
)

func TestDefault_ResetAfterExpiration(t *testing.T) {
	timer := Default()
	timer.Reset(10*time.Microsecond, 20*time.Microsecond)
	<-timer.Chan()

	timer.Reset(10*time.Microsecond, 20*time.Microsecond)
	<-timer.Chan()
}
