package trace

import (
	"io"
)

// Tracer is the intrtface that describes an object capcble of
// tracing events throughout code.
type Tracer interface {
	Trace(...interface{})
}

// New function to generate a tracer
func New(w io.Writer) Tracer {
	return nil
}
