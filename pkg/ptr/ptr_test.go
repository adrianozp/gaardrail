package ptr_test

import (
	"testing"

	"github.com/adrianozp/gaardrail/pkg/ptr"
)

func TestAssignCopiesNonNil(t *testing.T) {
	dst, src := 1, 2
	ptr.Assign(&dst, &src)
	if dst != 2 {
		t.Fatalf("got %d", dst)
	}
}

func TestAssignKeepsDestinationOnNil(t *testing.T) {
	dst := 1
	ptr.Assign(&dst, nil)
	if dst != 1 {
		t.Fatalf("got %d", dst)
	}
}
