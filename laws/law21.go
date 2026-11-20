package laws

import (
	"fmt"
	"math/rand"
	"time"
)

// LAW-21 Moves revoke fast (SB-804; sb-spaces(7), Moves): when the directory removes someone
// from a space's member group, an ask in that space is refused within 5 minutes; and a new
// membership grants nothing until the cell has confirmed it from the directory.

// MoveSystem is a cell with spaces and a directory feed, on a fake clock.
type MoveSystem interface {
	// Space makes a space whose members are group; Join puts user in group, confirmed.
	Space(name, group string) error
	Join(user, group string) error
	// Leave is the directory removing user from group at the current time.
	Leave(user, group string) error
	// JoinUnconfirmed is a membership the cell hasn't confirmed with the directory yet.
	JoinUnconfirmed(user, group string) error
	Advance(d time.Duration)
	// Allowed says whether user may ask in space now.
	Allowed(user, space string) (bool, error)
}

// MoveWindow is LAW-21's bound.
const MoveWindow = 5 * time.Minute

// NewMoveSystem builds the system under test: SCIM moves through the request path
// (SB-1205). See law21moves.go.
var NewMoveSystem = func() MoveSystem { return newSCIMMoves() }

func law21(r *rand.Rand, sys MoveSystem) error {
	if err := sys.Space("people", "group:people-team"); err != nil {
		return err
	}
	users := []string{"usr_ann", "usr_bob", "usr_cy"}
	for _, u := range users {
		if err := sys.Join(u, "group:people-team"); err != nil {
			return err
		}
	}
	for _, u := range users {
		if ok, err := sys.Allowed(u, "people"); err != nil || !ok {
			return fmt.Errorf("a confirmed member %s can't ask in people (%v)", u, err)
		}
	}
	gone := users[r.Intn(len(users))]
	if err := sys.Leave(gone, "group:people-team"); err != nil {
		return err
	}
	sys.Advance(time.Duration(r.Intn(int(MoveWindow/time.Second))+1) * time.Second)
	sys.Advance(MoveWindow)
	if ok, err := sys.Allowed(gone, "people"); err != nil || ok {
		return fmt.Errorf("%s still asks in people more than %v after leaving its group (err %v)", gone, MoveWindow, err)
	}
	if err := sys.JoinUnconfirmed("usr_new", "group:people-team"); err != nil {
		return err
	}
	if ok, _ := sys.Allowed("usr_new", "people"); ok {
		return fmt.Errorf("an unconfirmed membership granted the people space")
	}
	return nil
}
