# Project vision

Read [VISION.md](VISION.md) before proposing or implementing changes. Use its goals and non-goals to guide decisions, and flag any conflicts explicitly.

# Frontend
- **Use `pnpm`, NEVER `npm`**
- **Prefer arrow syntx functions:**
  - DO: `const f = () => { ... }`
  - DONT: `function f() { ... }`
- **Utilize scoped styling:** Svelte scopes styles to the component files by default. Utilize this and avoid unnecessary classes that scope the component
  - DO: `section { ... }; input { ... }`
  - DONT: `.component__base { ... }; .component__input { ... }`
- **Locally import webawesome components when used:** In every file where a webawesome component is used (`<wa-...`) that component MUST be imported in that file.

## Agent skills

### Issue tracker

Issues live in Backlog.md. See `docs/agents/issue-tracker.md`.

### Triage labels

Backlog.md labels mapped to canonical triage roles. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context repo. See `docs/agents/domain.md`.
