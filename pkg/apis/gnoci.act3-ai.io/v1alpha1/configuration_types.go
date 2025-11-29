// Package v1alpha1 defines the v1alpha1 schema.
//
// +kubebuilder:object:generate=true
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// Configuration type is used to store a user's current configuration settings.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	ConfigurationSpec `json:",inline"`
}

// ConfigurationSpec is the actual configuration values.
type ConfigurationSpec struct {
	RegistryConfig RegistryConfig `json:"registryConfig,omitempty"`
}

// RegistryConfig holds the custom configuration data for registries and repositories.
type RegistryConfig struct {
	Registries map[string]Registry `json:"registries"`
}

// Registry contains the custom configuration for a registry.
type Registry struct {
	// PlainHTTP enables http endpoints.
	PlainHTTP bool `json:"plainHTTP,omitempty"`

	// NonCompliant indicates a registry is not OCI compliant.
	NonCompliant bool `json:"noncompliant,omitempty"`
}

// ConfigurationDefault defaults the fields in [Configuration].
func ConfigurationDefault(obj *Configuration) {
	// Default the TypeMeta
	obj.APIVersion = GroupVersion.String()
	obj.Kind = "Configuration"
}
