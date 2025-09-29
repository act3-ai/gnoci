#!/usr/bin/env bash
############################################################
# crd-ref-docs: Generates markdown documentation for CRDs
############################################################

src="${1%\/}"
dst="${2%\/}"

last="" # tracks last package
for i in "$src"/*/*/*_types.go; do
	dir=$(dirname "$i")
	# Skip packages repeats
	[[ "$last" == "$dir" ]] && continue
	last="$dir"

	# Parse the important part of the path
	groupversion=${dir#"$src"/}

	NO_COLOR=1 tool/crd-ref-docs \
 		--config=internal/gen/.crd-ref-docs.yaml \
		--renderer=markdown \
		--log-level=ERROR \
		--source-path="${src}/${groupversion}" \
		--output-path="${dst}/${groupversion}.md"
done
