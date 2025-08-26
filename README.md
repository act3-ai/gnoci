# Git o(n) OCI (gnoci)

Pronounced as [gnocchi](https://www.merriam-webster.com/dictionary/gnocchi).

This project has two main objectives:

1. Define a specification for storing Git repositories as OCI artifacts.
2. Implement a Git remote helper to facilitate conversions of local Git repositories into remote OCI artifacts.

## Git as OCI Specification

A full specification is in progress. Please refer to the prototype [data model](docs/design/oci-data-model.md).

## Git Remote Helper for OCI Registries

> [!WARNING]  
> `git-remote-oci` is in early stages of development. Bugs are present, and inefficiencies are known to exist.

`git-remote-oci` is a Git remote helper that implements a custom protocol for interacting with Git repositories stored in OCI compliant registries. It is designed to allow users to interact with `git` as they normally do in their day-to-day workflows, but use an OCI registry as remote storage.

`git-remote-oci` supports:

- Cloning
- Fetching/Pulling
- Pushing
- Evaluating remote references

## Purpose

Why use OCI registries as remote storage for Git repositories?

Existing tools, such as [Zarf](https://zarf.dev/) and the [ASCE Data Tool](https://github.com/act3-ai/data-tool), provide solutions for moving OCI images and artifacts across air-gapped environments. The primary use-case for the `oci` remote helper protocol is to efficiently transfer and store Git repositories in OCI registries to complement the air-gap capabilities of these tools.

For more information see the [project proposal](./docs/proposal/proposal.md).

## Documentation

The documentation for `git-remote-oci` is organized as follows:

- **[Quick Start Guide](docs/quick-start-guide.md)**: provides documentation of installing and configuring `git-remote-oci`.
- **[User Guide](docs/user-guide.md)**: provides usage examples.
- **[Data Model](docs/design/oci-data-model.md)**: defines the data model used by `git-remote-oci` to store Git repositories in OCI compliant registries.

## How to Contribute

- **[Developer Guide](docs/developer-guide.md)**: detailed guide for contributing to the Git Remote Helper for OCI Registries repository.

## Support

- **[Troubleshooting FAQ](docs/troubleshooting-faq.md)**: consult list of frequently asked questions and their answers.
