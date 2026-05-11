# System Domain Boundary

`system` is a logical domain, not a separate runtime service yet.

Current submodules that belong to this domain:

- `user`
- `rbac`
- `identitysource`
- `systemsetting`
- `auth`

## Boundary Rules

- user / role / identity-source / setting logic should not depend on `k8s` or `machine`
- shared code goes to `internal/platform/*`
- HTTP router may wire these modules together, but business logic should stay inside the owning module
- cross-domain reads should go through stable service methods, not direct handler coupling

## Future Refactor Direction

If the codebase keeps growing, the next safe step is:

- move these packages under a shared `system` folder
- keep API shape stable
- only after that consider splitting `platform-center-core` into a separate repository
