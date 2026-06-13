// Package seamtest holds the conformance suite every seam.Store implementation must pass.
//
// Identical in intent to prp's seamtest: both the file-backed and (later) DynamoDB-backed stores
// call RunConformance with a fresh, empty store, so business logic above the seam relies on
// identical behavior regardless of backend. No live AWS.
package seamtest

import (
	"context"
	"errors"
	"testing"

	"github.com/scttfrdmn/prism/pkg/seam"
)

// Record is the type the conformance suite stores — small and backend-agnostic; it stands in for
// any governance record (approval, project, budget).
type Record struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
	Value int    `json:"value"`
}

// NewStore builds a fresh, empty store for one subtest. Each call must be independent.
type NewStore func(t *testing.T) seam.Store[Record]

// Two scopes that share a tenant but differ by PI/grant — used to prove isolation.
const (
	ownerA = "curie"
	ownerB = "bohr"
)

var (
	scopeA = seam.Scope{Tenant: "harvard", Project: "prism", PI: ownerA, Grant: "nsf-123"}
	scopeB = seam.Scope{Tenant: "harvard", Project: "prism", PI: ownerB, Grant: "doe-456"}
)

// RunConformance runs every behavioral contract of seam.Store against the given factory.
func RunConformance(t *testing.T, newStore NewStore) {
	t.Helper()

	t.Run("PutThenGet", func(t *testing.T) {
		s := newStore(t)
		ctx := context.Background()
		want := Record{ID: "r1", Owner: ownerA, Value: 7}
		if err := s.Put(ctx, scopeA, "r1", want); err != nil {
			t.Fatalf("Put: %v", err)
		}
		got, err := s.Get(ctx, scopeA, "r1")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got != want {
			t.Fatalf("Get = %+v, want %+v", got, want)
		}
	})

	t.Run("GetMissingReturnsErrNotFound", func(t *testing.T) {
		s := newStore(t)
		_, err := s.Get(context.Background(), scopeA, "nope")
		if !errors.Is(err, seam.ErrNotFound) {
			t.Fatalf("Get missing err = %v, want ErrNotFound", err)
		}
	})

	t.Run("PutOverwrites", func(t *testing.T) {
		s := newStore(t)
		ctx := context.Background()
		_ = s.Put(ctx, scopeA, "r1", Record{ID: "r1", Value: 1})
		if err := s.Put(ctx, scopeA, "r1", Record{ID: "r1", Value: 2}); err != nil {
			t.Fatalf("Put overwrite: %v", err)
		}
		got, _ := s.Get(ctx, scopeA, "r1")
		if got.Value != 2 {
			t.Fatalf("after overwrite Value = %d, want 2", got.Value)
		}
	})

	t.Run("DeleteThenGetMissing", func(t *testing.T) {
		s := newStore(t)
		ctx := context.Background()
		_ = s.Put(ctx, scopeA, "r1", Record{ID: "r1"})
		if err := s.Delete(ctx, scopeA, "r1"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(ctx, scopeA, "r1"); !errors.Is(err, seam.ErrNotFound) {
			t.Fatalf("Get after Delete err = %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteMissingReturnsErrNotFound", func(t *testing.T) {
		s := newStore(t)
		err := s.Delete(context.Background(), scopeA, "nope")
		if !errors.Is(err, seam.ErrNotFound) {
			t.Fatalf("Delete missing err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListEmptyIsEmptyNotNilError", func(t *testing.T) {
		s := newStore(t)
		got, err := s.List(context.Background(), scopeA)
		if err != nil {
			t.Fatalf("List empty: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("List empty len = %d, want 0", len(got))
		}
	})

	t.Run("ListReturnsAllInScope", func(t *testing.T) {
		s := newStore(t)
		ctx := context.Background()
		_ = s.Put(ctx, scopeA, "r1", Record{ID: "r1"})
		_ = s.Put(ctx, scopeA, "r2", Record{ID: "r2"})
		_ = s.Put(ctx, scopeA, "r3", Record{ID: "r3"})
		got, err := s.List(ctx, scopeA)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("List len = %d, want 3", len(got))
		}
		seen := map[string]bool{}
		for _, r := range got {
			seen[r.ID] = true
		}
		for _, id := range []string{"r1", "r2", "r3"} {
			if !seen[id] {
				t.Fatalf("List missing %q; got %+v", id, got)
			}
		}
	})

	// The load-bearing multi-tenant contract: records in one scope are invisible to another, even
	// when the same id is reused. This is the property Prism's single-user code lacks today.
	t.Run("ScopesAreIsolated", func(t *testing.T) {
		s := newStore(t)
		ctx := context.Background()
		if err := s.Put(ctx, scopeA, "shared-id", Record{ID: "shared-id", Owner: ownerA, Value: 1}); err != nil {
			t.Fatalf("Put scopeA: %v", err)
		}
		if err := s.Put(ctx, scopeB, "shared-id", Record{ID: "shared-id", Owner: ownerB, Value: 2}); err != nil {
			t.Fatalf("Put scopeB: %v", err)
		}

		gotA, err := s.Get(ctx, scopeA, "shared-id")
		if err != nil || gotA.Owner != ownerA || gotA.Value != 1 {
			t.Fatalf("scopeA Get = %+v, %v; want {curie,1}", gotA, err)
		}
		gotB, err := s.Get(ctx, scopeB, "shared-id")
		if err != nil || gotB.Owner != ownerB || gotB.Value != 2 {
			t.Fatalf("scopeB Get = %+v, %v; want {bohr,2}", gotB, err)
		}

		listA, _ := s.List(ctx, scopeA)
		if len(listA) != 1 {
			t.Fatalf("scopeA List len = %d, want 1 (leak from scopeB?)", len(listA))
		}
		if err := s.Delete(ctx, scopeA, "shared-id"); err != nil {
			t.Fatalf("Delete scopeA: %v", err)
		}
		if _, err := s.Get(ctx, scopeB, "shared-id"); err != nil {
			t.Fatalf("scopeB record vanished after scopeA delete: %v", err)
		}
	})
}
