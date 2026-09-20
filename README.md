# Terraform Provider for WSL

The Terraform WSL provider is a plugin that allows
[Terraform](https://www.terraform.io) to manage the lifecycle of Windows
Subsystem for Linux (WSL) distribution registrations on a Windows host.

## Quick Starts

- [Getting Started with the WSL Provider](docs/guides/getting-started.md)
- [Provider Documentation](docs/index.md)

## Provider Usage

Please see the [provider documentation](docs/index.md) for requirements,
configuration, and resource reference.

### Upgrading the provider

This provider doesn't upgrade automatically once you've started using it.
After a new release, run

```bash
terraform init -upgrade
```

to upgrade to the latest version allowed by your `required_providers`
version constraint. See the [Terraform
website](https://www.terraform.io/docs/configuration/providers.html#provider-versions)
for more on provider version constraints.

## Developing the provider

See [CONTRIBUTING.md](CONTRIBUTING.md) for the build/test setup and
acceptance tests, and [RELEASING.md](RELEASING.md) for the maintainer
release process.

## License

[Apache License 2.0](LICENSE).
