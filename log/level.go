package log

// Level defines an enumeration for a log message's level of importance.
type Level int

// Constant values of enumerated type Level.
const (
	Info Level = iota
	Error
)

// Symbol returns a rune representing the receiver Level; intended for use in
// log messages.
func (lev Level) Symbol() rune {
	return []rune(" !")[int(lev)]
}
