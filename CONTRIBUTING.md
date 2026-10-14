# Contributing to the Kaiten Terraform Provider

Thank you for contributing to the Kaiten Terraform Provider.

This project is licensed under the Apache License, Version 2.0.

## Developer Certificate of Origin

Kaiten uses the Developer Certificate of Origin 1.1 (DCO) for contributions.

Every commit must include a `Signed-off-by` trailer matching the commit author.

Use:

```bash
git commit -s -m "Describe your change"
```

This produces:

```text
Signed-off-by: Jane Doe <jane@example.com>
```

`git commit -s` is a **DCO sign-off**: it certifies the contribution under
[DCO.md](./DCO.md). `git commit -S`, with a capital S, is a **cryptographic
commit signature** made with a GPG or SSH key. They are not the same thing, and
the sign-off is what this project requires. There is nothing to install.

### Fixing a missing sign-off

A check named **DCO** verifies every commit of every pull request, and it is the
final word. If it fails, it names the commits to fix.

For the latest commit:

```bash
git commit --amend --signoff --no-edit
git push --force-with-lease
```

For several commits, sign off every commit of your branch at once:

```bash
git rebase --signoff origin/main
git push --force-with-lease
```

## Contribution rules

Please:

- keep pull requests focused;
- add or update tests when behavior changes;
- update documentation when relevant;
- do not include secrets, customer data, or confidential information;
- do not submit code or assets that you do not have the right to contribute;
- preserve required third-party license and attribution notices.

Accepted contributions are contributed under Apache-2.0.

## Changing the provider

The [README](./README.md#development) covers the toolchain and the tasks. Before opening a
pull request:

- run `task lint` and `task test`;
- run `task generate` after any change to a schema, an example or a template, and commit the
  regenerated `docs/` — CI fails on stale documentation;
- for a new or changed resource or data source, add acceptance tests (`TestAcc...`) and run
  them with `task testacc` against a disposable Kaiten organization;
- add an entry to [CHANGELOG.md](./CHANGELOG.md) for any user-facing change.

The provider follows HashiCorp's
[provider design principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles):
one resource per API object, read-only and side-effect-free data sources, and a resource that
removes itself from state when the object is gone.

## Code of conduct

This project follows the
[Kaiten code of conduct](https://github.com/kaitencloud/kaiten/blob/main/CODE_OF_CONDUCT.md).

## Security

Do not report unpatched vulnerabilities in public issues.

See [SECURITY.md](./SECURITY.md).
