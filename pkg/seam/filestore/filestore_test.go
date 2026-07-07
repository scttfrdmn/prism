package filestore_test

import (
	"testing"

	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/seam/seamtest"
)

func TestFilestore_Conformance(t *testing.T) {
	seamtest.RunConformance(t, func(t *testing.T) seam.Store[seamtest.Record] {
		return filestore.New[seamtest.Record](t.TempDir())
	})
}
