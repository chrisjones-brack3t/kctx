# kctx

`kctx` switches Kubernetes contexts and namespaces in the right order. It runs `kubectl` to select a context first, then sets that context's namespace.

## Requirements

- `kubectl` on your PATH and a working kubeconfig.
- Go 1.21 or later, if installing with Go.

## Install

Install with Go after a version is published:

```sh
go install github.com/chrisjones-brack3t/kctx@latest
```

Or download the binary for your system from the [releases page](https://github.com/chrisjones-brack3t/kctx/releases). Binaries are built for macOS and Linux, on `amd64` and `arm64`. Make the downloaded file executable and place it on your PATH, for example:

```sh
chmod +x kctx-macos-arm64
sudo mv kctx-macos-arm64 /usr/local/bin/kctx
```

Choose the file matching your operating system and processor: `amd64` for Intel or AMD, and `arm64` for Apple Silicon or ARM. The example installs the macOS Apple Silicon binary.

## Usage

Choose both a context and a namespace:

```sh
kctx -c dev-gke -n loon
kctx --context prod-gke --namespace content-container-service
```

Use the context name from your kubeconfig. Short and long flags are equivalent: `-c` or `--context`, and `-n` or `--namespace`.

Change the namespace of the current context:

```sh
kctx -n loon
```

Switch contexts while keeping the target context's saved namespace:

```sh
kctx -c prod-gke
```

Check the installed version or see the available flags:

```sh
kctx --version
kctx --help
```

Release binaries and Go-installed versions report their version tag. Local builds report the version supplied by Go, or `dev` when no version is available.

At least one context or namespace flag is required. Empty values and using both forms of the same flag are rejected. Kubernetes stores the namespace on the context, so changing it affects later `kubectl` commands using that context. If setting the namespace fails after a context switch, the context may already have changed; check the error before continuing.

## Releases

Push a version tag such as `v1.0.0` to build and publish the four binaries to a GitHub release. The workflow updates the assets if the release already exists.

## License

[MIT](LICENSE)
