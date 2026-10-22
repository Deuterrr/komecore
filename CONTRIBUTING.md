# Contributing

We'd love your help making KomeCore the cleanest, most reliable modular monolith e-commerce core in Go!

If you'd like to add new endpoints, domain services, or exported APIs, please [open an issue][open-issue] describing your proposal &mdash; discussing architectural and API changes ahead of time makes pull request review much smoother. In your issue, pull request, and any other communications, please remember to treat your fellow contributors with respect! We take our [code of conduct](CODE_OF_CONDUCT.md) seriously.

Looking for an easy place to get started? Explore issues tagged with [good first issue][good-first-issues].

## Setup

[Fork][fork], then clone the repository:

```bash
git clone git@github.com:your_github_username/komecore.git
cd komecore
git remote add upstream https://github.com/deuterrr/komecore.git
git fetch upstream
```

Initialize your local environment and launch the backing infrastructure (PostgreSQL 17 and Redis 7):

```bash
cp .env.example .env
docker compose up -d
make migrate
```

Make sure that the tests and quality gates pass:

```bash
make check
```

## Making Changes

Start by creating a new branch for your changes:

```bash
git checkout main
git fetch upstream
git rebase upstream/main
git checkout -b feat/my_cool_feature
```

Make your changes, then ensure that formatting, static analysis, and tests still pass:

```bash
make check
```

*(You can also run `make test` for fast in-memory unit tests or `make dev` for live reload with Air).*

If you're satisfied with your changes, push them to your fork:

```bash
git push origin feat/my_cool_feature
```

Then use the GitHub UI to open a pull request against `main`.

At this point, you're waiting on us to review your changes. We _try_ to respond to issues and pull requests within a few business days, and we may suggest some improvements or alternatives. Once your changes are approved and all CI checks pass, one of the project maintainers will merge them.

We're much more likely to approve your changes if you:

- Add tests for new functionality (`make test`, `make test-contract`, or `make test-integration`).
- Write a [good Conventional Commit message][commit-message] (e.g., `feat(order): add invoice download endpoint`).
- Maintain backward compatibility across existing API contracts and database schema migrations.
- Keep domain logic pure by isolating HTTP and SQL persistence from `internal/modules/<domain>`.

[fork]: https://github.com/deuterrr/komecore/fork
[open-issue]: https://github.com/deuterrr/komecore/issues/new
[good-first-issues]: https://github.com/deuterrr/komecore/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22
[commit-message]: https://www.conventionalcommits.org/
