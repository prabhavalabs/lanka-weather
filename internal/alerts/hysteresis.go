// Package alerts is the alert engine (ADR 0018). Every half minute, in the
// worker that holds the scheduler's lock, it reads our own tables through the
// services that serve the map, turns each subject (a gauge, a district's
// landslide level, a cyclone) into a level, and raises an event when a level
// rises above the one last alerted. Events go to push.Dispatcher.
//
// Hysteresis keeps a subject hovering at a threshold from alerting again and
// again: once alerted at a level, it alerts at that level or below again only
// after staying below it for the kind's hold time.
package alerts

import (
	"fmt"
	"time"
)

// State is what the engine remembers of a subject.
type State struct {
	Subject string
	Kind    string
	// Level is the level last observed.
	Level int
	// AlertedLevel is the level last alerted, lowered only after the hold.
	AlertedLevel int
	AlertedAt    *time.Time
	// BelowSince is when the level fell below AlertedLevel.
	BelowSince *time.Time
	UpdatedAt  time.Time
}

// Idle reports whether the state says nothing worth keeping.
func (state State) Idle() bool { return state.Level == 0 && state.AlertedLevel == 0 }

// EventKey identifies the episode an alert belongs to: the subject, its level
// and when that level was alerted. The same alert is never stored twice.
func (state State) EventKey() string {
	at := int64(0)
	if state.AlertedAt != nil {
		at = state.AlertedAt.Unix()
	}
	return fmt.Sprintf("%s:%d:%d", state.Subject, state.AlertedLevel, at)
}

// Step applies one observation to a subject's state. It returns the new state
// and whether to alert. A held observation (stale data, an unknown status)
// changes nothing: no data is neither a rise nor a fall.
func Step(previous State, level int, held bool, hold time.Duration, now time.Time) (State, bool) {
	if held {
		return previous, false
	}
	next := previous
	next.Level, next.UpdatedAt = level, now
	switch {
	case level > previous.AlertedLevel:
		at := now
		next.AlertedLevel, next.AlertedAt, next.BelowSince = level, &at, nil
		return next, true
	case level < previous.AlertedLevel:
		if previous.BelowSince == nil {
			since := now
			next.BelowSince = &since
		} else if now.Sub(*previous.BelowSince) >= hold {
			// Below long enough: the alerted level comes down to the level now,
			// so the next rise above it alerts.
			next.AlertedLevel, next.BelowSince = level, nil
			if level == 0 {
				next.AlertedAt = nil
			}
		}
	default:
		next.BelowSince = nil
	}
	return next, false
}
