package ethereum

import "errors"

// ErrNotFound is returned by API methods if the requested item does not exist.
var ErrNotFound = errors.New("not found")
