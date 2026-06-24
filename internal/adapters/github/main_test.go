package github

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// Ignore HTTP/2 transport goroutines left by net/http
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		goleak.IgnoreTopFunction("net/http.(*http2ClientConn).readLoop"),
		goleak.IgnoreTopFunction("net/http.(*http2Transport).newClientConn"),
	)
}
