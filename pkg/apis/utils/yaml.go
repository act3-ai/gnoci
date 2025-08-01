// Package utils provides miscellaneous utility functions.
package utils //nolint: revive

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ToYamlNodes converts v into a array of yaml.Nodes to be inserted into another document.
func ToYamlNodes(v any) ([]*yaml.Node, error) {
	node := &yaml.Node{}
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encoding yaml: %w", err)
	}
	err = yaml.Unmarshal(data, node)
	if err != nil {
		return nil, fmt.Errorf("decoding yaml: %w", err)
	}
	return node.Content, nil
}
