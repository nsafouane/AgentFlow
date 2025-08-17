package messaging

import (
    "reflect"
    "testing"
)

func TestCanonicalizeValue_MapAndStruct(t *testing.T) {
    s, err := NewCanonicalSerializer()
    if err != nil {
        t.Fatalf("failed to create serializer: %v", err)
    }

    // Map canonicalization
    m := map[string]interface{}{"b": 2, "a": 1}
    v := s.canonicalizeValue(m)
    if reflect.TypeOf(v).Kind() != reflect.Map {
        t.Fatalf("expected map result, got %T", v)
    }

    // Struct canonicalization
    type P struct {
        X int `json:"x"`
        Y int `json:"y"`
    }
    p := P{X: 10, Y: 20}
    sv := s.canonicalizeValue(p)
    if reflect.TypeOf(sv).Kind() != reflect.Map {
        t.Fatalf("expected struct -> map, got %T", sv)
    }

    // Slice/array canonicalization
    arr := []int{3, 2, 1}
    av := s.canonicalizeValue(arr)
    if reflect.TypeOf(av).Kind() != reflect.Slice {
        t.Fatalf("expected slice result, got %T", av)
    }
}
