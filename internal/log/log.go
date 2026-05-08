package log

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger = zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
}

// New returns a zerolog Logger that writes to w (or stderr if nil), with
// redaction applied to all string fields and messages.
func New(w io.Writer, level zerolog.Level) Logger {
	if w == nil {
		w = os.Stderr
	}
	return zerolog.New(redactedWriter{w}).
		Level(level).
		With().
		Timestamp().
		Logger()
}

type redactedWriter struct{ w io.Writer }

// Write returns len(p) (input length) by convention for transforming
// writers, even though the redacted output is shorter. zerolog tolerates
// this; bufio.Writer / io.MultiWriter wrappers may not — wrap with care.
func (r redactedWriter) Write(p []byte) (int, error) {
	out := []byte(Redact(string(p)))
	_, err := r.w.Write(out)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
