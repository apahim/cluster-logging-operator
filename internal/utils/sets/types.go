package sets

import (
	"sort"
)

type String struct {
	items map[string]struct{}
}

func NewString(values ...string) *String {
	s := &String{
		items: make(map[string]struct{}),
	}
	for _, v := range values {
		s.Insert(v)
	}
	return s
}

func (s *String) DeepCopyInto(in *String) {
	for item := range s.items {
		in.items[item] = struct{}{}
	}
}

func (s *String) Insert(values ...string) *String {
	for _, v := range values {
		s.items[v] = struct{}{}
	}
	return s
}

func (s *String) DeepCopy() *String {
	out := NewString()
	for item := range s.items {
		out.items[item] = struct{}{}
	}
	return out
}

func (s *String) List() []string {
	out := make([]string, 0, len(s.items))
	for item := range s.items {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func (s *String) Do(fn func(interface{})) {
	for item := range s.items {
		fn(item)
	}
}

func (s *String) Has(item string) bool {
	_, exists := s.items[item]
	return exists
}

func (s *String) Len() int {
	return len(s.items)
}

func (s *String) Remove(values ...string) *String {
	for _, v := range values {
		delete(s.items, v)
	}
	return s
}

// Set provides a generic set implementation for backwards compatibility
type Set struct {
	items map[interface{}]struct{}
}

func New() *Set {
	return &Set{
		items: make(map[interface{}]struct{}),
	}
}

func (s *Set) Insert(item interface{}) {
	s.items[item] = struct{}{}
}

func (s *Set) Has(item interface{}) bool {
	_, exists := s.items[item]
	return exists
}
