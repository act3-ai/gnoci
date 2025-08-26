// Package docs provides utilities for embeding documentation.
package docs

import (
	"embed"
	"io/fs"

	"github.com/spf13/cobra"

	"github.com/act3-ai/go-common/pkg/cmd"
	"github.com/act3-ai/go-common/pkg/embedutil"
)

// GeneralDocumentation is embedded general documentation.
//
//go:embed quick-start-guide.md
//go:embed user-guide.md
//go:embed troubleshooting-faq.md
var GeneralDocumentation embed.FS

// APIs is embedded API documentation.
//
//go:embed apis/gnoci.act3-ai.io/v1alpha1.md
var APIs embed.FS

//go:embed apis/schemas/*.schema.json
var schemas embed.FS

// Schemas returns the JSON Schema definitions.
func Schemas() fs.FS {
	filesys, err := fs.Sub(schemas, "apis/schemas")
	if err != nil {
		panic(err)
	}

	return filesys
}

// Embedded is a layout of embedded documentation to surface in the help command
// and generate in the gendocs command.
func Embedded(root *cobra.Command) *embedutil.Documentation {
	return &embedutil.Documentation{
		Title:   "Git Remote Helper for OCI Registries",
		Command: root,
		Categories: []*embedutil.Category{
			embedutil.NewCategory(
				"docs", "General Documentation", root.Name(), 1,
				embedutil.LoadMarkdown(
					"quick-start-guide",
					"Quick Start Guide",
					"quick-start-guide.md",
					GeneralDocumentation),
				embedutil.LoadMarkdown(
					"user-guide",
					"User Guide",
					"user-guide.md",
					GeneralDocumentation),
				embedutil.LoadMarkdown(
					"troubleshooting-faq",
					"Troubleshooting & FAQ",
					"troubleshooting-faq.md",
					GeneralDocumentation),
			),
			embedutil.NewCategory(
				"apis", "API Documentation", root.Name(), 5,
				embedutil.LoadMarkdown(
					"config-v1alpha1",
					"v1alpha1 API Documentation",
					"apis/gnoci.act3-ai.io/v1alpha1.md",
					APIs),
			),
		},
	}
}

// SchemaAssociations associates the schema file with all config file types.
var SchemaAssociations = []cmd.SchemaAssociation{
	{
		Definition: "gnoci.act3-ai.io.schema.json",
		// FileMatch:  actions.FileMatch,
	},
}
