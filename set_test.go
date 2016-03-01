package main

import "testing"

func TestSet(t *testing.T) {
	records := []record{
		{"id": 1},
		{"id": "2"},
	}

	set, err := NewSet("id", records)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := set.index["1"]; !ok {
		t.Error("expected 1 in index")
	}

	if _, ok := set.index["2"]; !ok {
		t.Error("expected 2 in index")
	}
}
