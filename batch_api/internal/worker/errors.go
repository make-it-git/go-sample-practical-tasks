package worker

import "errors"

var ErrNotQueued = errors.New("queue full")
