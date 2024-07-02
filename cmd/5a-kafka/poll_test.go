package main

import (
	"reflect"
	"testing"
)

func TestPoll(t *testing.T) {
	data := map[string][]int{
		"k1": {42, 43},
	}

	offsets := map[string]int{
		"k1": 0,
	}

	expected := map[string][][]int{
		"k1": {
			{0, 42},
			{1, 43},
		},
	}

	actual := poll(data, offsets)

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected: %v, actual: %v", expected, actual)
	}
}

// Tests that `poll` is copying its data, and not reusing slices
func TestPollConcurrent(t *testing.T) {
	data := map[string][]int{
		"k1": {42, 43},
	}

	offsets := map[string]int{
		"k1": 0,
	}

	for i := 0; i <= 10; i++ {
		go func() { poll(data, offsets) }()
	}

}
