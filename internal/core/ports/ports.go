// Package ports holds the interfaces the core depends on. Only interfaces and the
// plain data types they exchange live here; no vendor, no I/O (docs/06-engineering-standards.md).
// New ports arrive with the ticket that first needs them, never speculatively.
package ports

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// ConnectorAPIVersion is the version of the contract in sb-connector(3).
const ConnectorAPIVersion = 1

// ErrVersion means a connector implements a contract version the platform doesn't accept.
var ErrVersion = errors.New("ENOTSUP: connector API version not supported")

// Negotiate accepts a connector only if it implements a supported contract version
// (sb-connector(3), Version negotiation). There is no degraded mode.
func Negotiate(c Connector) error {
	if v := c.APIVersion(); v != ConnectorAPIVersion {
		return fmt.Errorf("%w: connector speaks v%d, platform speaks v%d", ErrVersion, v, ConnectorAPIVersion)
	}
	return nil
}

// SourceRef names one document in one source.
type SourceRef struct{ Kind, ID string }

// UserRef names the user an ACL question is about.
type UserRef struct{ ID, Email string }

// Document is what Fetch returns. ACLPrefilter is a stale-able copy, never an authority.
type Document struct {
	Ref          SourceRef
	Version      int
	Title, Text  string
	ACLPrefilter []string
}

// Verdict is a source's live answer. The zero value is VerdictUnknown, so a
// forgotten assignment refuses (LAW-03).
type Verdict int

const (
	VerdictUnknown Verdict = iota
	VerdictAllow
	VerdictDeny
)

func (v Verdict) String() string {
	switch v {
	case VerdictAllow:
		return "allow"
	case VerdictDeny:
		return "deny"
	case VerdictUnknown:
		return "unknown"
	}
	return "unknown"
}

// Cursor is an opaque, resumable sync position.
type Cursor string

// Change is one created, updated, deleted or permission-changed document.
type Change struct {
	Ref              SourceRef
	Deleted, ACLOnly bool
}

// ChangeSet is the result of one Changes call.
type ChangeSet []Change

// Action is a write for a gated workflow; Payload is canonical JSON.
type Action struct {
	Target  string
	Payload []byte
}

// IdempotencyKey is the approval id (sb-approvals(7)).
type IdempotencyKey string

// Receipt carries the target-issued id of a change (LAW-05).
type Receipt struct{ TargetID string }

// Connector is the four-call contract (sb-connector(3)).
type Connector interface {
	APIVersion() int
	Fetch(ctx context.Context, ref SourceRef) (Document, error)
	ACL(ctx context.Context, user UserRef, refs []SourceRef) ([]Verdict, error)
	Changes(ctx context.Context, cursor Cursor) (ChangeSet, Cursor, error)
	Act(ctx context.Context, a Action, key IdempotencyKey) (Receipt, error)
}

// Secrets resolves vault:// references at call time (LAW-07).
type Secrets interface {
	Get(ctx context.Context, ref string) (string, error)
}

// Clock is injected so time-bound laws (LAW-10, LAW-12) are testable.
type Clock interface {
	Now() time.Time
}

// Resource is one piece of a tenant's applied state, as listed in sb-tenant(1) (the plan).
type Resource struct {
	Kind  string            `json:"kind"`
	Name  string            `json:"name"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

// Registry is the control plane's record of what each tenant has applied (SB-102, SB-105).
// found is false only when the registry positively knows the tenant has no cell. An
// unreachable or unreadable registry is an error, never "no cell": that would plan a second cell.
type Registry interface {
	Applied(ctx context.Context, tenant string) (rs []Resource, found bool, err error)
}

// CellResource is one resource the cell template declares (sb-tenancy(7), Cell anatomy).
type CellResource struct {
	Kind, Name     string
	Tenant, Region string
}

// Placement creates or finds a cell's resources in one cloud account (ADR-0015, ADR-0018).
// Ensure is idempotent and returns the provider's id for the resource. If an adapter can't
// tell whether a resource exists, it returns an error; it never guesses "absent".
type Placement interface {
	Account() string
	Ensure(ctx context.Context, r CellResource) (id string, err error)
}

// Keys holds cell root keys and wraps data keys under them (envelope encryption), so a
// root key can be re-wrapped for a new custodian without re-encrypting data (LAW-18).
type Keys interface {
	EnsureRootKey(ctx context.Context, r CellResource) (keyID string, err error)
	Wrap(ctx context.Context, keyID string, dataKey []byte) ([]byte, error)
	Unwrap(ctx context.Context, keyID string, wrapped []byte) ([]byte, error)
}

// RegistryWriter records a tenant's applied state (SB-104). Put replaces the whole record.
type RegistryWriter interface {
	Put(ctx context.Context, tenant string, rs []Resource) error
}

// JournalEntry is one step of one apply run. It holds ids and names, never content.
type JournalEntry struct {
	Tenant  string `json:"tenant"`
	ApplyID string `json:"apply_id"`
	Step    int    `json:"step"`
	Op      string `json:"op"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Phase   string `json:"phase"` // "start" or "done"
}

// Journal is the write-ahead record of apply steps (sb-tenant(1)). A failed write stops apply.
type Journal interface {
	Record(ctx context.Context, e JournalEntry) error
}

// CellRecord is the registry's record of one cell (sb-tenancy(7), Registry and routing),
// including its custody record (ADR-0015).
type CellRecord struct {
	Tenant       string   `json:"tenant"`
	Region       string   `json:"region"`
	Endpoint     string   `json:"endpoint"`
	Hosts        []string `json:"hosts"`
	Custody      string   `json:"custody"`
	Placement    string   `json:"placement"`
	KeyCustodian string   `json:"key_custodian"`
}

// Cells finds cell records. found is false only when the registry positively knows there
// is no such cell; an unreachable registry is an error (unknown means refuse).
type Cells interface {
	CellByHost(ctx context.Context, host string) (c CellRecord, found bool, err error)
	CellByTenant(ctx context.Context, tenant string) (c CellRecord, found bool, err error)
}

// CellsWriter records a cell. A host may belong to only one cell.
type CellsWriter interface {
	PutCell(ctx context.Context, c CellRecord) error
}

// Identity is a verified sign-in: the tenant and user an ID token proves (sb-identity(7)).
type Identity struct {
	Tenant, Subject, Email, Issuer string
	IssuedAt                       time.Time
}

// IdentityProvider verifies one tenant's ID tokens. Every failure is an error: there is
// no partial or default identity.
type IdentityProvider interface {
	Verify(ctx context.Context, rawIDToken, nonce string) (Identity, error)
}

// Session is one signed-in user in one cell.
type Session struct {
	ID, Tenant, User string
	SignedIn         time.Time
	Validated        time.Time // last directory re-validation
}

// Sessions is a cell's own session store (SB-106).
type Sessions interface {
	Put(ctx context.Context, s Session) error
	Get(ctx context.Context, id string) (s Session, found bool, err error)
	Delete(ctx context.Context, id string) error
	RevokeUser(ctx context.Context, tenant, user string) (n int, err error)
}

// Directory answers whether a user is active, from SCIM-provisioned data (SB-107).
// An error, or a user it doesn't know, is never "active".
type Directory interface {
	Active(ctx context.Context, tenant, user string) (active, known bool, err error)
}

// User is a directory user, created only by SCIM (sb-identity(7)).
type User struct {
	ID, Tenant, UserName, ExternalID string
	Emails                           []string
	Active                           bool
}

// Group is a directory group; Members are user ids.
type Group struct {
	ID, Tenant, Name string
	Members          []string
}

// DirectoryStore is the write side of the directory, used only by the SCIM endpoint (SB-107).
type DirectoryStore interface {
	Directory
	PutUser(ctx context.Context, u User) error
	User(ctx context.Context, tenant, id string) (u User, found bool, err error)
	UserByName(ctx context.Context, tenant, userName string) (u User, found bool, err error)
	PutGroup(ctx context.Context, g Group) error
	Group(ctx context.Context, tenant, id string) (g Group, found bool, err error)
	DeleteGroup(ctx context.Context, tenant, id string) error
	// GroupsOf is what policy reads at decision time.
	GroupsOf(ctx context.Context, tenant, user string) ([]string, error)
}

// People is the scoped read side of the directory (SB-108): every method takes the
// caller's Scope and sees only that scope's tenant. An invalid Scope is an error.
type People interface {
	Person(ctx context.Context, s tenancy.Scope, id string) (u User, found bool, err error)
	MyGroups(ctx context.Context, s tenancy.Scope) ([]string, error)
}

// Chunk is one indexed span of one source document, with its provenance (sb-ingest(7)).
type Chunk struct {
	ID, Tenant, Cite, Kind, SourceID string
	Version                          int
	Start, End                       int // rune offsets in the normalised text
	Text, ContentSHA256              string
	Flags, ACLPrefilter              []string
	Release                          string
	Vector                           []float32
}

// Embedder turns chunk texts into vectors (SB-206). Model names the embedding model.
type Embedder interface {
	Model() string
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Index stores a tenant's chunks (SB-206). Replace swaps all of one source document's
// chunks in one write; the index is derived data and can be dropped and rebuilt (LAW-13).
type Index interface {
	Indexed(ctx context.Context, tenant string, ref SourceRef) (version int, found bool, err error)
	Replace(ctx context.Context, tenant string, ref SourceRef, version int, chunks []Chunk) error
	Delete(ctx context.Context, tenant string, ref SourceRef) error
	All(ctx context.Context, tenant string) ([]Chunk, error)
	Drop(ctx context.Context, tenant string) error
}

// CursorStore keeps each connector's sync cursor (SB-208). found is false only when no
// cursor was ever stored; an unreadable store is an error.
type CursorStore interface {
	Cursor(ctx context.Context, tenant, connector string) (c Cursor, found bool, err error)
	SaveCursor(ctx context.Context, tenant, connector string, c Cursor) error
}

// Query is one retrieval request: the text and its embedding.
type Query struct {
	Text   string
	Vector []float32
	Limit  int      // per ranking (vector, keyword); 0 means the default
	Kinds  []string // connector kinds to keep; empty means all (sb-ask(3), sources)
}

// Candidate is a retrieved chunk with its rank in each ranking (0 = not ranked there).
type Candidate struct {
	Chunk                   Chunk
	VectorRank, KeywordRank int
}

// Retriever searches one tenant's index (SB-301). The prefilter is applied inside the
// search: only chunks whose ACLPrefilter shares a token with tokens are candidates.
// A scope of another tenant is an error.
type Retriever interface {
	Search(ctx context.Context, s tenancy.Scope, tokens []string, q Query) ([]Candidate, error)
}

// Message is one turn of a model conversation.
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// ModelRequest is one chat completion (ADR-0019). System is the system prompt; JSON asks
// the provider for a JSON object where it supports that.
type ModelRequest struct {
	Model     string
	System    string
	Messages  []Message
	MaxTokens int
	JSON      bool
}

// Usage is the token count of one completion.
type Usage struct{ Input, Output int }

// ModelResponse is the model's text and usage.
type ModelResponse struct {
	Text  string
	Usage Usage
}

// Model errors every adapter maps to.
var (
	ErrRateLimited = errors.New("model: rate limited")
	ErrModelDown   = errors.New("model: unavailable")
)

// Model is one provider's chat completion (SB-305). Adapters never retry; the gateway does.
type Model interface {
	Complete(ctx context.Context, req ModelRequest) (ModelResponse, error)
}

// UserFinder maps an email to directory users (SB-308, chat identity). It returns every
// user of the tenant with that email, so callers can refuse anything but exactly one.
type UserFinder interface {
	UsersByEmail(ctx context.Context, tenant, email string) ([]User, error)
}

// EvalRecord is the latest eval run that may enable a workflow (sb-tenant(1), SB-410).
type EvalRecord struct {
	Passed bool      `json:"passed"`
	At     time.Time `json:"at"`
	Suites []string  `json:"suites"`
	Model  string    `json:"model"`
}

// EvalGate reads a tenant's eval record for one workflow.
type EvalGate interface {
	EvalRecord(ctx context.Context, tenant, workflow string) (r EvalRecord, found bool, err error)
}
