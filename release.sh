#!/usr/bin/env bash

# For custom changes, see https://daggerverse.dev/mod/github.com/act3-ai/dagger/release for dagger release module usage.

# Custom Variables
version_path="VERSION"
changelog_path="CHANGELOG.md"
notes_dir="releases"


# Dagger module dependencies, see dagger.json for versions
mod_release="release"
mod_gitcliff="git-cliff"
mod_goreleaser="goreleaser"


help() {
    cat <<EOF

Name:
    release.sh - Run a release process in stages.

Usage:
    release.sh COMMAND [-f | --force] [-i | --interactive] [-s | --silent]  [--version VERSION] [-h | --help]

Commands:
    prepare - prepare a release locally by running linters, tests, and producing the changelog, notes, assets, etc.

    approve - commit and tag your approved release.

    publish - push tag and publish the release to a remote by uploading assets, images, helm chart, etc.

Options:
    -h, --help
        Prints usage and other helpful information.

    -i, --interactive
        Run the release process interactively, prompting for approval to continue for each stage: prepare, approve, and publish. By default it begins with the prepare stage, otherwise it "resumes" the process at a specified stage. Alternatively, set \$INTERACTIVE.

    -s, --silent
        Run dagger silently, e.g. 'dagger --silent'. Alternatively, set \$SILENT.

    -f, --force
       Skip git status checks, e.g. uncommitted changes, in all stages and linters in prepare. Alternatively, set \$FORCE.

    --version VERSION
        Run the release process for a specific semver version, ignoring git-cliff's configured bumping strategy. Alternatively, set \$VERSION.

Required Environment Variables:
    - GITHUB_API_TOKEN     - repo:api access
    - RELEASE_AUTHOR       - username of release author, for homebrew tap
    - RELEASE_AUTHOR_EMAIL - email of release author, for homebrew tap
    - SSH_PRIVATE_KEY      - ssh key for homebrew tap

Dependencies:
    - dagger
    - git
EOF
    exit 1
}

# insufficient args
if [ "$#" -eq 0 ]; then
    help
fi

set -euo pipefail

# Defaults. Overriden by flag equivalents, when applicable.
cmd=""
force="${FORCE:-false}"       # skip git status checks
interactive="${INTERACTIVE:-false}" # interactive mode
silent="${SILENT:-false}"      # silence dagger (dagger --silent)
explicit_version="${VERSION:-""}"  # release for a specific version

release_latest="${RELEASE_LATEST:-false}" # tag release as latest


# Get commands and flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    # Commands
    "prepare" | "approve" | "publish")
       cmd=$1
       shift
       ;;
    # Flags
    "-h" | "--help")
       help
       ;;
    "--version")
       shift
       explicit_version=$1
       shift
       ;;
    "-i" | "--interactive")
       interactive=true
       shift
       ;;
    "-s" | "--silent")
       silent=true
       shift
       ;;
    "-f" | "--force")
       force=true
       shift
       ;;
    *)
       echo "Unknown option: $1"
       help
       ;;
  esac
done

# Interactive mode begins with prepare by default, otherwise continue the release
# process at the specified stage. Must occur after parsing commands and flags, else
# we risk unexpected behavior, e.g. 'release.sh -f' would imply prepare.
if [ "$interactive" = "true" ] && [ -z "$cmd" ]; then
    cmd="prepare"
fi

# prompt_continue requests user input until a valid y/n option is provided.
# Inputs:
#   - $1 : name of next stage to continue to.
# disable read without -r backslash mangling for this func
# shellcheck disable=SC2162
prompt_continue() {
    read -p "Continue to $1 stage (y/n)?" choice
    case "$choice" in
    y|Y )
        echo -n "true"
    ;;
    n|N )
        echo -n "false"
        ;;
    * )
        echo "Invalid input '$choice'" >&2
        prompt_continue "$1"
        ;;
    esac
}

# check_upstream ensures remote upstream matches local commit.
# Inputs:
#  - $1 : commit, often HEAD or HEAD~1
check_upstream() {
    if [ "$force" != "true" ]; then
        echo "Comparing local $1 to remote upstream"
        git diff "@{upstream}" "$1" --stat --exit-code
    fi
}

# prepare runs linters and unit tests, bumps the version, and generates the changelog.
# runs 'approve' if interactive mode is enabled.
prepare() {
    echo "Running prepare stage..."

    # old_version=v$(cat "$version_path")
    
    # linters and unit tests
    if [ "$force" != "true" ]; then
        dagger -m="$mod_release" -s="$silent" --src="." call \
            go check
    fi
    git fetch --tags
    check_upstream "HEAD"

    # bump version, generate changelogs
    vVersion=""
    if [ "$explicit_version" != "" ]; then
        vVersion="$explicit_version"
    else
        vVersion=$(dagger -m="$mod_gitcliff" -s="$silent" --src="." call bumped-version)
    fi

    # verify release version with gorelease
    # if [ "$force" != "true" ]; then
    #     dagger -m="$mod_release" -s="$silent" --src="." call \
    #         go verify --target-version="$vVersion" --current-version="$old_version"
    # fi

    dagger -m="$mod_release" -s="$silent" --src="." call prepare \
    --ignore-error="$force" \
    --version="$vVersion" \
    --version-path="$version_path" \
    --changelog-path="$changelog_path" \
    export --path="."

    echo -e "Successfully ran prepare stage.\n"
    echo -e "Please review the local changes, especially releases/$vVersion.md\n"
    if [ "$interactive" = "true" ] && [ "$(prompt_continue "approve")" = "true" ]; then
            approve
    fi
}

# approve commits changes and adds a release tag locally.
# runs 'publish' if interactive mode is enabled.
approve() {
    echo "Running approve stage..."

    git fetch --tags
    check_upstream "HEAD"

    vVersion=v$(cat "$version_path")
    notesPath="${notes_dir}/${vVersion}.md"

    # stage release material
    git add "$version_path" "$changelog_path" "$notesPath"
    git add \*.md
    
    # signed commit
    git commit -S -m "chore(release): prepare for $vVersion"
    # annotated and signed tag
    git tag -s -a -m "Official release $vVersion" "$vVersion"

    echo -e "Successfully ran approve stage.\n"
    if [ "$interactive" = "true" ] && [ "$(prompt_continue "publish")" = "true" ]; then
            publish
    fi
}

# publish pushes the release tag, uploads release assets, and publishes images.
publish() {
    echo "Running publish stage..."

    git fetch --tags
    check_upstream "HEAD~1" # compare before our release commit, i.e. we're only fast forwarding that commit

    # push this branch and the associated tags
    git push --follow-tags

    vVersion=v$(cat "$version_path")

    dagger -m="$mod_goreleaser" -s="$silent" --src="." call \
    with-secret-variable --name="GITHUB_TOKEN" --secret=env:GITHUB_API_TOKEN \
    with-secret-variable --name="SSH_PRIVATE_KEY" --secret=env:SSH_PRIVATE_KEY \
    with-env-variable --name="RELEASE_LATEST" --value="$release_latest" \
    with-env-variable --name="RELEASE_AUTHOR" --value="$RELEASE_AUTHOR" \
    with-env-variable --name="RELEASE_AUTHOR_EMAIL" --value="$RELEASE_AUTHOR_EMAIL" \
    release \
    with-notes --notes "${notes_dir}/${vVersion}.md" \
    run

    echo -e "Successfully ran publish stage.\n"
    echo "Release process complete."
}

# Run the release script.
case $cmd in
"prepare")
    prepare
    ;;
"approve")
    approve
    ;;
"publish")
    publish
    ;;
*)
    help
    ;;
esac