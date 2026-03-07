package marketplace

import "time"

type PackageMeta struct {
	Name        string    `json:"name" yaml:"name"`
	Version     string    `json:"version" yaml:"version"`
	Description string    `json:"description" yaml:"description"`
	Author      string    `json:"author" yaml:"author"`
	Category    string    `json:"category" yaml:"category"`
	Difficulty  string    `json:"difficulty" yaml:"difficulty"`
	Tags        []string  `json:"tags" yaml:"tags"`
	Chains      []string  `json:"chains" yaml:"chains"`
	Services    []string  `json:"services" yaml:"services"`
	License     string    `json:"license" yaml:"license"`
	Premium     bool      `json:"premium" yaml:"premium"`
	Price       string    `json:"price" yaml:"price"`
	Downloads   int       `json:"downloads" yaml:"downloads"`
	Stars       int       `json:"stars" yaml:"stars"`
	Homepage    string    `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Repository  string    `json:"repository,omitempty" yaml:"repository,omitempty"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

type SearchResult struct {
	Query    string         `json:"query"`
	Total    int            `json:"total"`
	Packages []PackageMeta  `json:"packages"`
}

type InstalledPackage struct {
	PackageMeta `json:",inline" yaml:",inline"`
	InstalledAt time.Time `json:"installed_at" yaml:"installed_at"`
	Path        string    `json:"path" yaml:"path"`
}
