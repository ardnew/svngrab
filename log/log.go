package log

import (
	"fmt"
	"io"
)

// Log represents an object for writing log messages.
// All messages are written to the io.Writer member given to its initializer
// function.
type Log struct{ output io.Writer }

// New initializes and returns a pointer to a new Log.
func New(output io.Writer) *Log {
	return &Log{output: output}
}

// Break writes a single newline sequence to the receiver's io.Writer based on
// the current host system (i.e., Unix: LF/0xA, Windows: CR+LF/0xD+0xA).
func (l *Log) Break() {
	fmt.Fprint(l.output, Eol)
}

// Putf prints to the receiver's io.Writer a string described by the given
// format string and list of arguments.
// No decorators or line-endings are placed anywhere around this string; it is
// printed to the stream verbatim.
func (l *Log) Putf(format string, args ...interface{}) {
	fmt.Fprintf(l.output, format, args...)
}

// Writef prints to the receiver's io.Writer a single line consisting of:
//
//   a. Log level symbol (indicating "info" or "error", for example);
//   b. A logical class or group to which the message belongs; and
//   c. The log message itself.
//
// Note that a newline is not appended to the message by default. This provides
// a way to dynamically append stateful info -- such as results from a
// long-running operation -- onto the message once operation completes.
//
// For example, the following output can be recreated using this design:
//    "   [download] host/url -> myPath ..." (** 60s elapses **) "ok!\n"
func (l *Log) Writef(level Level, class string, format string, args ...interface{}) {
	fmt.Fprintf(l.output, " %c [%s] ", level.Symbol(), class)
	l.Putf(format, args...)
}

// Infof calls Writef by automatically using Info for level.
// All other arguments are passed through to Writef as-is.
func (l *Log) Infof(class string, format string, args ...interface{}) {
	l.Writef(Info, class, format, args...)
}

// Errorf calls Writef by automatically using Error for level.
// All other arguments are passed through to Writef as-is.
func (l *Log) Errorf(class string, format string, args ...interface{}) {
	l.Writef(Error, class, format, args...)
}

// Eolf calls Putf and Break to append the given format and args to the current
// line, and then calls Errorf with the given error if it is non-nil.
// All other arguments are passed through to Writef as-is.
func (l *Log) Eolf(class string, err error, format string, args ...interface{}) {
	if nil == err {
		l.Putf(format, args...)
	}
	l.Break()
	if nil != err {
		l.Errorf(class, "%s", err.Error())
		l.Break()
	}
}
