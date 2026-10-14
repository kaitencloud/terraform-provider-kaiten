# Terraform Provider for Kaiten

The Kaiten provider manages [Kaiten](https://kaiten.sh) infrastructure with Terraform. It
covers **deployment zones**, the regions and environments that instances are deployed to:

| Type | Name |
| --- | --- |
| Resource | `kaiten_deployment_zone` |
| Data source | `kaiten_deployment_zone` |
| Data source | `kaiten_deployment_zones` |

It talks to the Kaiten Core API through the official Go SDK,
[`github.com/kaitencloud/sdk-go`](https://github.com/kaitencloud/sdk-go), and is built on the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).

## Usage

```terraform
terraform {
  required_providers {
    kaiten = {
      source = "kaitencloud/kaiten"
    }
  }
}

# Reads KAITEN_BASE_URL and KAITEN_AUTH_TOKEN, or the Kaiten CLI configuration file.
provider "kaiten" {}

resource "kaiten_deployment_zone" "eu_west" {
  name = "AWS eu-west-1"
  type = "production"

  metadata = jsonencode({
    region = "eu-west-1"
  })
}
```

The provider resolves its base URL and API token from the provider block, then the
`KAITEN_BASE_URL` and `KAITEN_AUTH_TOKEN` environment variables, then the configuration file the
[Kaiten CLI](https://github.com/kaitencloud/cli) writes. Full documentation is in [`docs/`](docs/),
generated from the provider schema and [`examples/`](examples/).

## Development

Requirements: [Go](https://go.dev/doc/install) 1.26+, [Terraform](https://developer.hashicorp.com/terraform/install)
1.5+, and [Task](https://taskfile.dev).

```sh
task build      # go build ./...
task test       # unit tests
task lint       # golangci-lint
task generate   # format examples/ and regenerate docs/ -- commit the result
```

### Acceptance tests

Acceptance tests create and delete real deployment zones. Point them at a disposable
organization:

```sh
export KAITEN_BASE_URL=https://kaiten.example.com/api
export KAITEN_AUTH_TOKEN=ksh_...
task testacc                         # all of them
task testacc -- -run TestAccDeploymentZone_basic
```

### Running a local build

Install the binary with `task install`, then point Terraform at it in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "kaitencloud/kaiten" = "/path/to/your/GOBIN"
  }
  direct {}
}
```

### Layout

```
main.go                     provider server; go:generate for the docs
internal/provider/
  provider.go               provider schema and Configure
  settings.go               base URL and token resolution
  deployment_zone.go        model and mapping shared by the resource and data sources
  deployment_zone_*.go      kaiten_deployment_zone resource and data source
  deployment_zones_*.go     kaiten_deployment_zones data source
examples/                   HCL pulled into the docs by tfplugindocs
templates/                  docs templates; docs/ is generated from them
```

### Releasing

Push a `vX.Y.Z` tag from `main`. The release workflow runs GoReleaser, which builds the archives,
the signed `SHA256SUMS` and the manifest the Terraform Registry ingests. It needs the
`GPG_PRIVATE_KEY` and `PASSPHRASE` secrets, and the key's public half registered with the
Registry. Update `CHANGELOG.md` first.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Every commit must be signed off (`git commit -s`).
Report vulnerabilities as described in [SECURITY.md](SECURITY.md).

## License

The Kaiten Terraform Provider is open source and licensed under the
[Apache License, Version 2.0](./LICENSE).

By contributing, you agree to certify your contribution under the
[Developer Certificate of Origin 1.1](./DCO.md).

The Kaiten name and logos are not licensed under Apache-2.0. See the
[Kaiten trademark policy](https://github.com/kaitencloud/kaiten/blob/main/TRADEMARKS.md).
