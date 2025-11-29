// Package modelmock mocks pkg model.
package modelmock

//go:generate go tool mockgen -typed -package modelmock -destination ./modelmock.gen.go github.com/act3-ai/gnoci/internal/model ReadOnlyModeler,Modeler
