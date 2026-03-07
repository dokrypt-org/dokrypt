package types

type ServiceStatus struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Status  string         `json:"status"`
	Ports   map[string]int `json:"ports,omitempty"`
	URLs    map[string]string `json:"urls,omitempty"`
	Healthy bool           `json:"healthy"`
}
