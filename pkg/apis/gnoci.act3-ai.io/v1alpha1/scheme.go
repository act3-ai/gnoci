// Package v1alpha1 defines the v1alpha1 schema.
//
// +kubebuilder:object:generate=true
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Group is the API group.
	Group = "gnoci.act3-ai.io"
	// Version is the group API version.
	Version = "v1alpha1"
	// Repository is the location of this project.
	Repository = "github.com/act3-ai/gnoci"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Adds the list of known types to the given scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		GroupVersion,
		&Configuration{},
	)
	scheme.AddTypeDefaultingFunc(&Configuration{}, func(in any) { ConfigurationDefault(in.(*Configuration)) })
	return nil
}
