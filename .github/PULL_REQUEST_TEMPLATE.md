<!--
Keep the title short (<= 70 chars) and present tense:
  feat(scope): add X     fix(scope): handle Y     docs(scope): clarify Z
-->

## What & why

<!-- One paragraph: what does this change, and what constraint forced it?
     Skip the "what" if the diff is self-evident. Always include the why. -->

## Checklist

- [ ] `make ci-local` passes locally (build / vet / test / lint / fmt-check)
- [ ] `CHANGELOG.md` Unreleased section updated, OR change is purely internal
- [ ] New `pkg/` code has at least one behavioral test
- [ ] No business logic added to `pkg/`

## Test plan

<!-- How did you verify this works? E.g.:
       - unit tests added: pkg/foo/foo_test.go
       - `make test-integration` against the dev stack
       - manual: ran `kris-alpha`, curl-tested /version
-->
