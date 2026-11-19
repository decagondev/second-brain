---
title: Release inputs
summary: The pinned model and how a bundle is built.
---

# Release inputs

[`model.json`](model.json) pins the model the release runs with (sb-release(7), Bundles). The `local-dev` value is for development bundles; a real release sets the provider's model id here, and that change is a new release id like any other.

Build a bundle:

```sh
go build -o bin/sb ./cmd/sb
go run ./tools/release/bundle --artefact sb=bin/sb --created 2027-03-02 --out bundle.json
```
