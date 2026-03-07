package testrunner

import (
	"context"
)

type Suite struct {
	Name        string
	Description string
	Setup       func(ctx context.Context) error // Run before suite
	Teardown    func(ctx context.Context) error // Run after suite
	Tests       []TestCase
}

type TestCase struct {
	Name string
	Fn   func(ctx context.Context) error
	Tags []string
}

func NewSuite(name string) *Suite {
	return &Suite{Name: name}
}

func (s *Suite) AddTest(name string, fn func(ctx context.Context) error) {
	s.Tests = append(s.Tests, TestCase{Name: name, Fn: fn})
}

func (s *Suite) AddTaggedTest(name string, tags []string, fn func(ctx context.Context) error) {
	s.Tests = append(s.Tests, TestCase{Name: name, Fn: fn, Tags: tags})
}
