package procguard

import (
	"time"
)

//
// Process describes a process'es command-line, exec result etc.
//
type RunStat struct {
	Start  time.Time
	Finish time.Time
	Error  error
}
