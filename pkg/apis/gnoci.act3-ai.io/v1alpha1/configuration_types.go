// Package v1alpha1 defines the v1alpha1 schema.
//
// +kubebuilder:object:generate=true
package v1alpha1

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiutils "github.com/act3-ai/gnoci/pkg/apis/utils"

	"github.com/act3-ai/go-common/pkg/redact"

	"github.com/act3-ai/go-common/pkg/logger"
)

// +kubebuilder:object:root=true

// Configuration type is used to store a user's current configuration settings.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	ConfigurationSpec `json:",inline"`
}

// ConfigurationSpec is the actual configuration values.
type ConfigurationSpec struct {
	// Example description for ExampleOption
	ExampleOption bool `json:"exampleOption,omitempty"`

	// Name is your name
	Name string `json:"name"`
}

// ConfigurationDefault the fields in Configuration.  The argument must be a Configuration.
func ConfigurationDefault(obj *Configuration) {
	if obj == nil {
		obj = &Configuration{}
	}

	// Default the TypeMeta
	obj.APIVersion = GroupVersion.String()
	obj.Kind = "Configuration"

	// This is called after we decode the values (from file) so we need to be careful not to overwrite values that are already set.
	// We can use pointers if we need to know that a value has been set or not.
	if obj.Name == "" {
		obj.Name = "None"
	}
}

// Redacted implements the logr.Marshaller interface.
func (c *ConfigurationSpec) Redacted() *ConfigurationSpec {
	retval := c.DeepCopy()

	retval.Name = redact.String(retval.Name)

	return retval
}

// MarshalLog implements the logr.Marshaller interface.
func (c *Configuration) MarshalLog() any {
	// remove TypeMeta so that this retval does not conform to fmt.Stringer interface
	return c.ConfigurationSpec.MarshalLog()
}

// MarshalLog implements the logr.Marshaller interface.
func (c *ConfigurationSpec) MarshalLog() any {
	return c.Redacted()
}

// Write writes the config file in the given folder.
func (c *Configuration) Write(ctx context.Context, path string) error {
	yamlContent, err := c.ToDocumentedYAML(ctx)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, yamlContent, 0o644)
	if err != nil {
		return fmt.Errorf("could not write config file %q: %w", path, err)
	}

	return nil
}

// ToDocumentedYAML converts a Configuration into YAML with comments explaining each field.
func (c Configuration) ToDocumentedYAML(ctx context.Context) ([]byte, error) {
	log := logger.FromContext(ctx)

	// create a top level yaml.Node.  Note, the document node is already created by
	//  this point, so the top level is a key value mapping node.
	nodes := []*yaml.Node{
		{
			Kind:  yaml.ScalarNode,
			Value: "kind",
		},
		{Kind: yaml.ScalarNode, Value: "Configuration"},
		{
			Kind:  yaml.ScalarNode,
			Value: "apiVersion",
		},
		{Kind: yaml.ScalarNode, Value: GroupVersion.String()},
	}

	addField := func(name, header, footer string, subNodes []*yaml.Node, empty bool) {
		if !empty {
			footer = ""
		}
		nodes = append(nodes, &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       name,
			HeadComment: "\n" + header,
			FootComment: footer,
		})
		nodes = append(nodes, subNodes...)
	}

	subNodes, err := apiutils.ToYamlNodes(c.Name)
	if err != nil {
		return nil, fmt.Errorf("unable to parse configuration: %w", err)
	}
	addField("name", "Your name", "", subNodes, c.Name == "")

	subNodes, err = apiutils.ToYamlNodes(c.ExampleOption)
	if err != nil {
		return nil, fmt.Errorf("unable to parse configuration: %w", err)
	}
	addField("exampleOption", "Example option", "", subNodes, false)

	doc := &yaml.Node{
		Kind:        yaml.DocumentNode,
		HeadComment: commentConfigHead,
		FootComment: commentConfigFoot,
		Content: []*yaml.Node{
			{
				Kind:    yaml.MappingNode,
				Content: nodes,
			},
		},
	}

	yamlContent, err := yaml.Marshal(doc)
	if err != nil {
		log.InfoContext(ctx, "could not marshal Configuration", "error", err)
		return nil, fmt.Errorf("unable to parse configuration: %w", err)
	}

	return yamlContent, nil
}

const (
	commentConfigHead = `git-remote-oci Configuration File
Stores configuration for git-remote-oci`
	commentConfigFoot = "Comments added by the user will not be preserved in this file"
)
