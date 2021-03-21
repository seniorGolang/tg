package types

// Base type for all (almost) entities.
// It contains name of entity and docs.
// Docs is a comments in golang syntax above entity declaration.
// Each block comment is counted as one.
type Base struct {
	Name string   `json:"name,omitempty"`
	Docs []string `json:"docs,omitempty"`
}
