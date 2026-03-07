package state

import (
	"crypto/sha256"
	"fmt"
	"time"
)

func NewSnapshot(name string, project string, opts SaveOptions) *Snapshot {
	return &Snapshot{
		Name:        name,
		Description: opts.Description,
		Tags:        opts.Tags,
		CreatedAt:   time.Now().UTC(),
		Project:     project,
		Chains:      make(map[string]ChainSnapshot),
	}
}

func (s *Snapshot) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (s *Snapshot) AddTag(tag string) {
	if !s.HasTag(tag) {
		s.Tags = append(s.Tags, tag)
	}
}

func (s *Snapshot) Age() time.Duration {
	return time.Since(s.CreatedAt)
}

func ConfigHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}
