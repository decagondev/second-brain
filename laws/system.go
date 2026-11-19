package laws

import "fmt"

// Citation points at one document version.
type Citation struct {
	Doc     string
	Version int
}

// Answer is what an ask returns: what the model was given and what it cited.
type Answer struct {
	Context   []string
	Citations []Citation
	Claims    []ComposedClaim // the composed answer, when the system composes one
}

// ToolCall is one proposed tool call.
type ToolCall struct {
	ID            string
	Consequential bool
	Approval      string // "" when none was given
}

// Outcome is the reported result of a tool call.
type Outcome struct {
	State   string // "Ran", "DidNotRun" or "Refused"
	Receipt string
}

// System is the surface the laws test. Real wiring replaces NewSystem as the
// tickets land: Ask with SB-302, Call with SB-405 and SB-408.
type System interface {
	Ask(w *World, userID, question string) (Answer, error)
	Call(w *World, userID string, tc ToolCall) (Outcome, error)
}

// NewSystem builds the system under test for a world.
var NewSystem = func(*World) System { return pending{} }

type pending struct{}

func (pending) Ask(*World, string, string) (Answer, error) {
	return Answer{}, fmt.Errorf("%w: the ask pipeline arrives with SB-302", ErrPending)
}

func (pending) Call(*World, string, ToolCall) (Outcome, error) {
	return Outcome{}, fmt.Errorf("%w: the tool bus arrives with SB-405 and SB-408", ErrPending)
}
