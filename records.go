package main

import (
	"errors"
	"fmt"
	"reflect"
)

type record map[string]interface{}

type Set struct {
	key     string
	records []record
	index   map[string]record
}

var (
	ErrDuplicateItem     = errors.New("duplication item found for set")
	ErrInvalidIdentifier = errors.New("identifier is not a colum")
)

func NewSet(key string, records []record) (*Set, error) {
	s := &Set{
		key:     key,
		records: records,
	}

	s.index = make(map[string]record, len(records))

	var (
		idv interface{}
		id  string
	)

	for _, r := range records {
		idv = r[key]

		if idv == nil {
			return nil, ErrInvalidIdentifier
		}

		switch x := idv.(type) {
		case string:
			id = x
		default:
			return nil, fmt.Errorf("bad identifier type. expected string, got %T", idv)
		}

		if _, ok := s.index[id]; ok {
			return nil, ErrDuplicateItem
		}

		s.index[id] = r
	}

	return s, nil
}

func (s *Set) Compare(m *SetMetrics, b *Set) {
	m.Lock()
	defer m.Unlock()

	m.Size.Set(float64(len(b.records)))

	var (
		ok     bool
		ak, bk string
		av, bv interface{}
	)

	for ak, av = range s.index {
		if bv, ok = b.index[ak]; !ok {
			m.Removals.Inc()
			continue
		}

		if !reflect.DeepEqual(av, bv) {
			m.Changes.Inc()
		}
	}

	for bk, _ = range b.index {
		if _, ok = s.index[bk]; !ok {
			m.Additions.Inc()
		}
	}
}
