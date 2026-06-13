// Package seam defines the record/persistence + identity interface Prism's governance managers
// converge onto (design §5). It is the SHARED seam: byte-identical in contract to
// prism-research-portal's pkg/seam, so a record written by one is read by the other.
//
// This compatibility IS the shared-state guarantee (design §4): Prism (desktop) and prp (web)
// read and write the SAME records, under the SAME partition keys, in the SAME backing store —
// never two parallel stores reconciled by a sync protocol. A record prp writes lands in the exact
// partition Prism reads, and vice versa; live propagation rides the existing control-plane path.
//
// Two halves:
//
//   - Principal — the identity half: the generalization of spore-host's
//     Principal{Project, AccountID}, extended with tenant / pi / grant / affiliation.
//   - Store[T] — the persistence half: a scoped repository, implemented twice (file-backed for
//     the desktop, DynamoDB-backed for the cloud core). Business logic above the seam never knows
//     which it is talking to — which is what keeps Prism usable standalone (filestore) while the
//     same managers serve the shared cloud (dynamostore).
//
// The tenancy key IS the identity: Scope is an alias for Principal. Every read and write is scoped.
//
// NOTE: this file is intentionally kept identical to prism-research-portal/pkg/seam/seam.go. prp
// will re-point at THIS definition (deleting its duplicate) as the unification step; until then,
// the two must not drift — the shared-state guarantee depends on it.
package seam

import (
	"context"
	"errors"
	"strings"
)

// ErrNotFound is returned by Store.Get / Store.Delete when no record exists for (scope, id).
// Implementations MUST return this exact error (use errors.Is) so callers can branch on it
// without depending on a backend-specific type.
var ErrNotFound = errors.New("seam: record not found")

// Principal is a validated caller identity. It extends spore-host's
// Principal{Project, AccountID} with the institutional dimensions prp adds. The string fields map
// one-to-one onto the prp:* ABAC tag namespace (tenant/pi/grant/affiliation).
type Principal struct {
	Tenant      string // institution / org unit       → prp:tenant
	Project     string // existing spore.host dimension
	PI          string // principal investigator        → prp:pi
	Grant       string // funding source                → prp:grant
	Affiliation string // eduPerson affiliation → tier   → prp:affiliation
	AccountID   string // member / compute account
}

// Scope is the persistence tenancy key. It is the Principal: a record is partitioned by who owns
// it. Backends derive a partition key from Scope (see Principal.Key).
type Scope = Principal

// Key is the deterministic partition key for a scope: tenant/project/pi/grant joined in a fixed
// order. Empty dimensions collapse to a single separator, so a Scope with only Tenant set still
// produces a stable, distinct key. AccountID and Affiliation are NOT part of the key — Affiliation
// is a derived tier, and a record's home does not move when acted on from a different account.
//
// This MUST stay identical to prp's Principal.Key(): the partition key is the wire contract that
// makes the same record reachable from both clients.
func (p Principal) Key() string {
	return strings.Join([]string{p.Tenant, p.Project, p.PI, p.Grant}, "/")
}

// Store is a scoped repository for records of type T. It is the persistence half of the seam.
// Implementations: filestore (single local tree, desktop-standalone) and dynamostore (the shared
// cloud core). All methods are scoped: an id is only unique within a Scope. Get and Delete on a
// missing record return ErrNotFound. List returns an empty slice (never nil-with-error) when a
// scope holds no records.
type Store[T any] interface {
	Get(ctx context.Context, scope Scope, id string) (T, error)
	List(ctx context.Context, scope Scope) ([]T, error)
	Put(ctx context.Context, scope Scope, id string, v T) error
	Delete(ctx context.Context, scope Scope, id string) error
}
