package sim

import "fmt"

// EventLevel is the severity of a log entry.
type EventLevel uint8

const (
	Info EventLevel = iota
	Warn
	Err
)

// Event is one line in the city log.
type Event struct {
	Day   int
	Level EventLevel
	Msg   string
}

// MaxEvents is the size of the log ring.
const MaxEvents = 200

// Logf appends an event, dropping the oldest past MaxEvents.
func (c *City) Logf(level EventLevel, format string, args ...any) {
	c.Log = append(c.Log, Event{Day: c.Day, Level: level, Msg: fmt.Sprintf(format, args...)})
	if len(c.Log) > MaxEvents {
		c.Log = append(c.Log[:0], c.Log[len(c.Log)-MaxEvents:]...)
	}
}

var months = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// DaysPerMonth and MonthsPerYear define the calendar.
const (
	DaysPerMonth  = 30
	MonthsPerYear = 12
)

// Date formats a day count as "Mar 14, Y3".
func Date(day int) string {
	m := (day / DaysPerMonth) % MonthsPerYear
	y := day/(DaysPerMonth*MonthsPerYear) + 1
	return fmt.Sprintf("%s %d, Y%d", months[m], day%DaysPerMonth+1, y)
}
