# util

`util` only contains small, domain-neutral helpers that are easier to use than
the standard library directly.

Keep these packages generic where Go supports it:

- `convutil`: scalar conversion through `To[T]` and `ToOr[T]`.
- `sliceutil`, `maputil`, `ptrutil`, `mathutil`: typed collection, pointer, and numeric helpers.
- `env`: typed environment reads through `Get[T]`.
- `validator`: composable generic validation rules.
- `codec`, `serializer`: focused byte/string encoding and JSON helpers.
- `retry`, `syncutil`, `ctxutil`: focused runtime helpers.

Do not add thin wrappers around standard packages unless they remove meaningful
boilerplate. Prefer direct use of packages such as `slices`, `maps`,
`context`, `strings`, and `strconv` when the wrapper would only rename one
function.

Avoid adding framework-specific helpers under `util`. Existing Gin response
helpers remain in `serverresp` for now; do not move them into the root `http`
module without an explicit module-boundary change.
