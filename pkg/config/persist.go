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

type Persister struct {
	mu   sync.Mutex
	path string
}

func NewPersister(cfg Config) *Persister {
	return &Persister{path: cfg.Path}
}

func (p *Persister) Set(updates map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.path == "" {
		return fmt.Errorf("config: no config file path to persist to")
	}
	doc, err := p.read()
	if err != nil {
		return err
	}
	root := documentRoot(doc)
	for path, val := range updates {
		setScalar(root, strings.Split(path, "."), val)
	}
	return p.write(doc)
}

func (p *Persister) read() (*yaml.Node, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", p.path, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", p.path, err)
	}
	return &doc, nil
}

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

func (p *Persister) write(doc *yaml.Node) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("config: marshal %s: %w", p.path, err)
	}
	_ = enc.Close()
	if err := os.WriteFile(p.path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("config: write %s: %w", p.path, err)
	}
	return nil
}

func setScalar(mapping *yaml.Node, keys []string, val any) {
	ensureMapping(mapping)
	next := childValue(mapping, keys[0])
	if len(keys) == 1 {
		encodeScalar(next, val)
		return
	}
	setScalar(next, keys[1:], val)
}

func ensureMapping(n *yaml.Node) {
	if n.Kind == yaml.MappingNode {
		return
	}
	n.Kind = yaml.MappingNode
	n.Tag = "!!map"
	n.Value = ""
	n.Content = nil
}

func childValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valNode := &yaml.Node{}
	mapping.Content = append(mapping.Content, keyNode, valNode)
	return valNode
}

func encodeScalar(n *yaml.Node, val any) {
	n.Kind = yaml.ScalarNode
	n.Content = nil
	switch v := val.(type) {
	case string:
		n.Tag = "!!str"
		n.Style = yaml.DoubleQuotedStyle
		n.Value = v
	case float64:
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
