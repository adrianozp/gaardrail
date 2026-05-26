package config

import (
	"bytes"
	"errors"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ControllerTypeStore persists the selected controller type back to the config
// file. It edits only the controller.type key in place, preserving the rest of
// the file (comments, ordering, untouched keys). Crucially it does NOT use
// viper.WriteConfig, which would also bake transient env-var overrides
// (APP_*) into the file.
type ControllerTypeStore struct{}

func NewControllerTypeStore() ControllerTypeStore { return ControllerTypeStore{} }

func (ControllerTypeStore) PersistControllerType(t string) error {
	path := viper.ConfigFileUsed()
	if path == "" {
		return errors.New("config: no config file in use, cannot persist controller type")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	setMappingValue(rootMapping(&doc), "controller", func(controller *yaml.Node) {
		setScalar(controller, "type", t)
	})

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) // match the repo's existing 2-space style
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// rootMapping returns the top-level mapping node, creating the document/mapping
// structure if the file was empty.
func rootMapping(doc *yaml.Node) *yaml.Node {
	if len(doc.Content) == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	return doc.Content[0]
}

// setMappingValue finds the mapping nested under key (creating it if absent) and
// applies edit to it.
func setMappingValue(parent *yaml.Node, key string, edit func(*yaml.Node)) {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			edit(parent.Content[i+1])
			return
		}
	}
	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		child,
	)
	edit(child)
}

// setScalar sets key=value (string) in mapping, updating in place or appending.
func setScalar(mapping *yaml.Node, key, value string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].Kind = yaml.ScalarNode
			mapping.Content[i+1].Tag = "!!str"
			mapping.Content[i+1].Value = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
