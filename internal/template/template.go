package template

type Template struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description"`
	Author      string   `yaml:"author" json:"author"`
	Tags        []string `yaml:"tags" json:"tags"`
	Chains      []string `yaml:"chains" json:"chains"`
	Services    []string `yaml:"services" json:"services"`
	License     string   `yaml:"license" json:"license"`
	Category    string   `yaml:"category" json:"category"`
	Difficulty  string   `yaml:"difficulty" json:"difficulty"`
	Premium     bool     `yaml:"premium" json:"premium"`
	Price       string   `yaml:"price" json:"price"`
}

type Vars struct {
	ProjectName string
	ChainName   string
	ChainID     uint64
	Engine      string
	Author      string
}

type Info struct {
	Template Template `json:"template"`
	Path     string   `json:"path"`
	BuiltIn  bool     `json:"built_in"`
}
