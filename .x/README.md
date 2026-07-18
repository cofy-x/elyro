# Elyro Repository Context

This index is the only repository context document that must be read for every task after the root [Agent contract](../AGENTS.md). Select the narrowest matching row and load only that context; read the [validation matrix](validation.md) before final verification.

## Task Routing

| Change scope | Required context |
| :--- | :--- |
| Root layout, path ownership, root Make logic, release layout, or structural moves | [Repository ownership map](project-overview.md) |
| `internal/workspace`, Workspace behavior, Workspace wiring under `cmd/elyro`, starters, examples, or Workspace scripts | [Workspace ownership and invariants](project-overview.md#workspace) |
| `skills/`, `elyro skill`, or host coding-agent guidance | [Repository ownership map](project-overview.md) and [integration topology and routing](integration-stack.md) |
| Shared or root implementation such as `internal/cliui`, `internal/images`, `internal/version`, or `cmd/elyro` behavior not owned by Workspace | [Repository ownership map](project-overview.md) plus the affected user or image documentation |
| Supported image definitions or image layering | [Supported image catalog](../images/README.md) and the owning image guide; add the [ownership map](project-overview.md) when layout or ownership changes |
| CI, release configuration, repository scripts, or workflow orchestration | [Repository ownership map](project-overview.md) and the invoking Make target or script documentation |
| Product direction, support scope, or a new workflow family | [Product roadmap](../docs/roadmap.md) plus the affected ownership context |
| Final verification for any implementation or documentation change | [Validation matrix](validation.md) |

If no row matches, read the [repository ownership map](project-overview.md) before editing and update this router when the task introduces a new ownership category.

## Sources Of Truth

- [Repository ownership map](project-overview.md): root layout, domain invariants, package ownership, Make layers, and structural sync points.
- [Integration topology and routing](integration-stack.md): caller, Workspace, SSH, image, and Skill boundaries.
- [Validation matrix](validation.md): minimum checks, behavior checks, and repository hygiene.
- User guides under `docs/` and image READMEs: supported user behavior and operational commands.
- [Product roadmap](../docs/roadmap.md): product direction and scope.

Keep each fact in its owning document. Link to that source instead of copying its detailed rules into this index or another context file.
