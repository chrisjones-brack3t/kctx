# kctx

`kctx` switches Kubernetes contexts and namespaces in the right order. It runs `kubectl` to select a context first, then sets that context's namespace.

## Requirements

- `kubectl` on your PATH and a working kubeconfig.
- Kubernetes contexts named `dev-gke` and `prod-gke`.
- Go 1.21 or later, if installing with Go.

## Install

Install with Go after a version is published:

```sh
go install github.com/chrisjones-brack3t/kctx@latest
```

Or download the binary for your system from the [releases page](https://github.com/chrisjones-brack3t/kctx/releases). Binaries are built for macOS (`darwin`) and Linux, on `amd64` and `arm64`. Make the downloaded file executable and place it on your PATH, for example:

```sh
chmod +x kctx-darwin-arm64
sudo mv kctx-darwin-arm64 /usr/local/bin/kctx
```

Choose the file matching your operating system and processor. The example installs the macOS Apple Silicon binary.

## Usage

Choose both a context and a namespace:

```sh
kctx --context dev --namespace loon
kctx --context prod --namespace content-container-service
```

`dev` selects `dev-gke`; `prod` selects `prod-gke`.

Change only the namespace:

```sh
kctx --namespace loon
```

This uses the current context, which must be `dev-gke` or `prod-gke`.

Change only the context:

```sh
kctx --context prod
```

This carries the current context's namespace to `prod-gke`. If the current context has no namespace set, `kctx` uses `default`. The current context can have any name for this operation.

See the available flags:

```sh
kctx --help
```

At least one flag is required. Empty namespaces and context values other than `dev` and `prod` are rejected. Kubernetes stores the namespace on the context, so changing it affects later `kubectl` commands using that context. If setting the namespace fails, the context may already have switched; check the error before continuing.

## Releases

Push a version tag such as `v1.0.0` to build and publish the four binaries to a GitHub release. The workflow updates the assets if the release already exists.

## License

[MIT](LICENSE)
