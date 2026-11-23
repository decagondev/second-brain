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
	// Location is where the document lives at the source, for routing to spaces, and Label
	// the source's own sensitivity label, "" for none (sb-connector(3); SB-1005).
	Location, Label string
	// Owner is the email of the document's owner at the source, "" for none (SB-1203).
	Owner string
	// Fields are a structured document's data by field name (SB-1106); ingest builds the
	// text from the allowed ones. nil for a document that is text.
	Fields map[string]string
	// ReviewBy is a review or expiry date the source keeps apart from the text, as written
	// (an Office file's custom review-by property; SB-2705); ingest checks it. "" for none.
	ReviewBy string
}

// Field is one field a structured connector can return (sb-connector(3), Fields).
type Field struct {
	Name      string
	Sensitive bool // pay, performance, cases, personal identifiers, bank details, tax ids
}

// FieldDeclarer is a connector that returns fields. ok is false when it can't say what it
// returns; then none of its fields is ingested.
type FieldDeclarer interface {
	Fields() (fields []Field, ok bool)
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
	// Token is a read-only API token's public id (tok_…), with its name and end (SB-2306);
	// "" is a browser session. A token's ID is the SHA-256 of the token, never the token.
	Token   string    `json:",omitempty"`
	Name    string    `json:",omitempty"`
	Expires time.Time `json:",omitzero"`
}

// Sessions is a cell's own session store (SB-106).
type Sessions interface {
	Put(ctx context.Context, s Session) error
	Get(ctx context.Context, id string) (s Session, found bool, err error)
	Delete(ctx context.Context, id string) error
	RevokeUser(ctx context.Context, tenant, user string) (n int, err error)
	// ListUser is every session of one user, browser and token (SB-2306).
	ListUser(ctx context.Context, tenant, user string) ([]Session, error)
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
	// Manager is the user id of their manager, from SCIM's enterprise extension, or "" (SB-1204).
	Manager string
	// From SCIM (SB-1205): the core title and the enterprise extension's organisation facts.
	Title, Department, Division, CostCenter string
}

// Attributes are a user's directory attributes as policy sees them (sb-policy(5)); empty
// ones are left out.
func (u User) Attributes() map[string]string {
	out := map[string]string{}
	for k, v := range map[string]string{"department": u.Department, "division": u.Division, "cost_center": u.CostCenter,
		"title": u.Title, "manager": u.Manager} {
		if v != "" {
			out[k] = v
		}
	}
	return out
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
	// In a cell with spaces (SB-1005): the label, the ids of the spaces that hold the
	// document, and the fingerprint of the space configuration that routed it.
	Label   string
	Spaces  []string
	Routing string
	// Owner is the document's owner at the source, an email or "" (SB-1203): barriers
	// check the owner's groups, from the directory at decision time.
	Owner string
	// Title is the document's title as indexed, and Seen when the index first saw this
	// version (zero when it doesn't know) (SB-1701).
	Title string
	Seen  time.Time
	// Location is the document's location at the source (Document.Location), for start
	// lists and handover briefs (SB-1706, SB-1707).
	Location string
	// Rebuilt marks a version a rebuild first saw, whose Seen is the rebuild's time, not
	// when it changed: "what changed" never lists one (SB-1708).
	Rebuilt bool
	// TextSHA256 is the SHA-256 of the document's normalised text, without its title, the
	// same on every chunk of the version, "" for no text: copies share it (SB-2202).
	TextSHA256 string
	// ReviewBy is the document's own review-by or expires date (SB-2705); zero for none.
	// A cite is stale from it, if that is sooner than its age makes it.
	ReviewBy time.Time
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

// CiteLookup is an index that finds chunks by their citation (doc@version), for a cite a
// person added, such as a steward's reply (SB-1704); found chunks still take the live check.
type CiteLookup interface {
	ChunksOf(ctx context.Context, tenant string, cites []string) ([]Chunk, error)
}

// DocQuery picks indexed documents by source, by location under a folder, or by owner.
type DocQuery struct {
	Refs   []SourceRef
	Under  []SourceRef // ID is a location prefix ending in "/"
	Owners []string    // owner emails, lower case
	// SeenSince picks versions first seen at or after it, never rebuilt ones (SB-1708).
	SeenSince time.Time
	// Flag picks chunks carrying the flag, e.g. personal_data (SB-1904).
	Flag string
	// Space picks chunks a space holds, by id (spc_NAME) (SB-2201, SB-2202).
	Space string
}

// DocLookup is an index that finds chunks by DocQuery (SB-1706, SB-1707); found chunks
// still take every check before anyone sees them.
type DocLookup interface {
	Docs(ctx context.Context, tenant string, q DocQuery) ([]Chunk, error)
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
	// Also are phrases a chunk may match instead of Text: the meanings of the glossary
	// terms the query used (SB-2203). Vector is the embedding of Text with them.
	Also []string
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
	User      string // who the call is for (budget and rate limit); never sent to a provider
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
