package config

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Persister writes runtime config changes back to the loaded config file using
// targeted edits: only the addressed keys change, every other key, ordering and
// comment is preserved. viper.WriteConfig is deliberately avoided because it
// bakes resolved env overrides into the file.
type Persister struct {
	mu   sync.Mutex
	path string
}

func NewPersister(cfg Config) *Persister {
	return &Persister{path: cfg.Path}
}

// Set applies all updates in a single read-modify-write. Keys are dotted yaml
// paths (e.g. "queue.protocol"); missing intermediate maps and leaves are
// created. Values keep their natural yaml type (string/number).
func (p *Persister) Set(updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.path == "" {
		return fmt.Errorf("config: no config file path to persist to")
	}

	data, err := os.ReadFile(p.path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", p.path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("config: parse %s: %w", p.path, err)
	}

	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	for path, val := range updates {
		setScalar(root, strings.Split(path, "."), val)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("config: marshal %s: %w", p.path, err)
	}
	_ = enc.Close()

	if err := os.WriteFile(p.path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("config: write %s: %w", p.path, err)
	}
	return nil
}

// setScalar walks mapping nodes by keys, creating any missing node, and writes
// the leaf value. Existing nodes are mutated in place so their comments survive.
func setScalar(mapping *yaml.Node, keys []string, val any) {
	if mapping.Kind != yaml.MappingNode {
		mapping.Kind = yaml.MappingNode
		mapping.Tag = "!!map"
		mapping.Value = ""
		mapping.Content = nil
	}

	key := keys[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			next := mapping.Content[i+1]
			if len(keys) == 1 {
				encodeScalar(next, val)
				return
			}
			setScalar(next, keys[1:], val)
			return
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valNode := &yaml.Node{}
	mapping.Content = append(mapping.Content, keyNode, valNode)

	if len(keys) == 1 {
		encodeScalar(valNode, val)
		return
	}
	setScalar(valNode, keys[1:], val)
}

// encodeScalar sets a scalar value on a node, picking the yaml type from the Go
// type. Strings use double-quoted style so values with yaml-special characters
// (e.g. a colon in a SQL query) round-trip safely.
func encodeScalar(n *yaml.Node, val any) {
	n.Kind = yaml.ScalarNode
	n.Content = nil
	switch v := val.(type) {
	case string:
		n.Tag = "!!str"
		n.Style = yaml.DoubleQuotedStyle
		n.Value = v
	case float64:
		// Leave the tag empty so the emitter infers int/float from the value and
		// no explicit "!!float" tag is written.
		n.Tag = ""
		n.Style = 0
		n.Value = strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		n.Tag = ""
		n.Style = 0
		n.Value = strconv.Itoa(v)
	default:
		n.Tag = "!!str"
		n.Style = yaml.DoubleQuotedStyle
		n.Value = fmt.Sprint(v)
	}
}
