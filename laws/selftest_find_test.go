package laws

import (
	"math/rand"
	"testing"
)

// The W16 Find laws catch the systems they exist to catch (SB-1701 to SB-1703).

func failsSomewhere(t *testing.T, name string, prop func(*rand.Rand) error) {
	t.Helper()
	for seed := int64(1); seed <= 300; seed++ {
		if prop(rand.New(rand.NewSource(seed))) != nil {
			return
		}
	}
	t.Fatalf("%s: no seed caught the broken system", name)
}

func TestSelfFindSearchUnknownAsAllowIsCaught(t *testing.T) {
	failsSomewhere(t, "search with Unknown as Allow", func(r *rand.Rand) error { return law01Search(r, searchSystem{unknownAsAllow: true}) })
}

func TestSelfFindInactiveOwnerIsCaught(t *testing.T) {
	failsSomewhere(t, "owners ignoring deprovisioning", func(r *rand.Rand) error { return law12Owners(r, searchSystem{anyOwner: true}) })
}

func TestSelfFindTenantBlindSharesAreCaught(t *testing.T) {
	failsSomewhere(t, "shares not keyed by tenant", func(r *rand.Rand) error { return law02Shares(r, shareSystem{sharedStore: true}) })
}

func TestSelfAskStewardCitesWidenIsCaught(t *testing.T) {
	failsSomewhere(t, "asker shown every steward cite", func(r *rand.Rand) error { return law01StewardCites(r, questionSystem{stewardSeesAll: true}) })
}

func TestSelfAskAnyoneAnswersIsCaught(t *testing.T) {
	failsSomewhere(t, "steward rule ignoring groups", func(r *rand.Rand) error { return law21StewardQueue(r, questionSystem{anyoneAnswers: true}) })
}

func TestSelfStartUnknownAsAllowIsCaught(t *testing.T) {
	failsSomewhere(t, "start list with Unknown as Allow", func(r *rand.Rand) error { return law01StartHere(r, searchSystem{unknownAsAllow: true}) })
}

func TestSelfHandoverUnknownAsAllowIsCaught(t *testing.T) {
	failsSomewhere(t, "handover with Unknown as Allow", func(r *rand.Rand) error { return law12Handover(r, searchSystem{unknownAsAllow: true}) })
}

func TestSelfChangesForgottenSeenIsCaught(t *testing.T) {
	failsSomewhere(t, "a rebuild that forgets first-seen times", func(r *rand.Rand) error { return law13Changes(r, changeSystem{forgetSeen: true}) })
}
