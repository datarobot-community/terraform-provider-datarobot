<!-- Replace placeholders; keep checklist. -->

## Summary
Describe what this PR changes and why.

## Type
- [ ] feat (new feature)
- [ ] fix (bug fix)
- [ ] docs
- [ ] chore / refactor
- [ ] test
- [ ] ci

## Details / User Impact
Explain user-visible effects (provider behavior, new resources/data sources, breaking changes, deprecations).

## CHANGELOG
- [ ] Added an entry under Unreleased in CHANGELOG.md
- [ ] Not needed (label `skip changelog` applied and no user impact)
If skipped, justify here:

## Testing
Describe test coverage or add instructions.

## Release Preparation Checklist
Complete if this should go into next release:
- [ ] CHANGELOG updated
- [ ] All CI checks green
- [ ] Reviewers assigned / approved
- [ ] Version impact considered (patch / minor / major)

## After Merge (maintainers / releaser)
To cut a release:
```
git fetch origin
git checkout main
git pull
# choose next version: vX.Y.Z
git tag vX.Y.Z
git push origin vX.Y.Z
```
This triggers release workflow: .github/workflows/release.yml

## Additional Notes
Anything else reviewers should know.
