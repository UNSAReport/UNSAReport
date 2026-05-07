package capture

import "time"

type TimedInput struct {
	Delay time.Duration
	Text  string
}
