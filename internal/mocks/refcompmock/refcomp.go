// Package refcompmock mocks pkg refcomp.
package refcompmock

//go:generate go tool mockgen -typed -package refcompmock -destination ./refcompmock.gen.go github.com/act3-ai/gnoci/internal/refcompmock RefComparer
