package sliceutil

import (
	"reflect"
	"testing"
)

func TestFirstLast(t *testing.T) {
	if _, ok := First([]int{}); ok {
		t.Fatalf("First(empty) ok = true, want false")
	}
	if got := FirstOr([]int{}, 9); got != 9 {
		t.Fatalf("FirstOr(empty) = %d, want 9", got)
	}
	if got, ok := Last([]int{1, 2, 3}); !ok || got != 3 {
		t.Fatalf("Last = %d/%v, want 3/true", got, ok)
	}
}

func TestSetOperationsPreserveOrder(t *testing.T) {
	if got := Unique([]int{2, 1, 2, 3, 1}); !reflect.DeepEqual(got, []int{2, 1, 3}) {
		t.Fatalf("Unique = %#v", got)
	}
	if got := Intersect([]int{3, 1, 3, 2}, []int{1, 3}); !reflect.DeepEqual(got, []int{3, 1}) {
		t.Fatalf("Intersect = %#v", got)
	}
	if got := Union([]int{2, 1}, []int{1, 3}); !reflect.DeepEqual(got, []int{2, 1, 3}) {
		t.Fatalf("Union = %#v", got)
	}
}
