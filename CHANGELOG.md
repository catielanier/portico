# Changelog

All notable changes to Portico will be documented in this file.

Portico follows semantic versioning before 1.0 loosely: patch releases may still include internal refactors when they support bug fixes.

## [0.4.1] - 2026-09-10

### Fixed

- Fixed required license handling during package transactions.
  - Portico now parses package-scoped license requirements reported by `emerge --pretend`.
  - License requirements can be resolved for the requested atom or dependencies.
  - Required licenses are applied to the temporary Portage sandbox first.
  - Confirmed license entries are persisted to `/etc/portage/package.license/90-portico` only after the user approves the final transaction.
  - Portico no longer blocks transactions that can be resolved with explicit package-scoped license acceptance.

- Fixed USE flag selection for packages with no USE flags.
  - Packages with zero USE flags now skip the USE flag picker entirely.
  - Portico continues directly to the next applicable transaction step instead of showing an empty selector.

- Fixed USE flag selection for packages with large USE flag sets.
  - The USE flag picker now paginates when flags exceed the available terminal viewport.
  - Pagination is calculated dynamically from the current terminal height.
  - Selections are preserved while moving between pages.
  - Pagination recalculates when the terminal is resized.
  - Action buttons remain visible instead of being pushed below the viewport.

- Fixed dependency-resolution status output.
  - Dependency checks now display a single `Checking dependencies...` status instead of exposing internal retry numbers.
  - Portico can perform multiple resolver passes internally without showing noisy retry labels to the user.

- Improved keyword autounmask handling during dependency resolution.
  - Portico now parses required keyword changes reported by Portage autounmask output.
  - Keyword changes can be applied to the temporary Portage sandbox and retried.
  - This improves handling for keyword requirements that appear deeper in the dependency tree.
  - The previous suspected `accept_keywords` traversal issue is now treated as an investigation item rather than a confirmed bug.

- Fixed install progress reporting.
  - Progress now represents completed packages rather than the package currently being processed.
  - A transaction starts at `0%`.
  - Progress advances only after a package has completed successfully.
  - Failed, interrupted, or merely-started packages are not counted as completed work.
  - A successful transaction reaches `100%` only after the final package completes.

- Fixed repository synchronization using stale overlay state.
  - Portico now treats Gentoo’s currently enabled repository state as the source of truth.
  - Repositories removed outside Portico are no longer considered eligible for synchronization.
  - Stale directories under `/var/db/repos` no longer cause removed overlays to reappear as active repositories.
  - Stale Portico cache entries no longer cause removed repositories to be synchronized.
  - Repository metadata from `repos.conf`, repository directories, and Portico cache data is now supporting metadata only.
  - Removed overlays can no longer block unrelated installs, rebuilds, or updates during pre-transaction repository sync.

### Changed

- Added an animated install status spinner.
  - The install UI now shows active movement while Portage is compiling or installing packages.
  - This makes long source builds feel alive instead of frozen.

- Added a compilation-time notice to the install UI.
  - Portico now reminds users that installation may take some time due to compilation.

- Improved USE picker button behavior.
  - Single-page flag lists show `Confirm` and `Cancel`.
  - Multi-page flag lists show page-appropriate actions such as `Next`, `Prev`, `Confirm`, and `Cancel`.
  - `Next` is the primary action until the final page.
  - `Confirm` becomes the primary action on the final page.

- Clarified repository state ownership.
  - Portico may assist with repository workflows, but external Gentoo tools such as `eselect repository` remain authoritative.
  - Future Portico repository-management commands should remain compatible with repository changes made outside Portico.

### Internal

- Extended autounmask parsing to support:
  - USE changes
  - keyword changes
  - license changes

- Consolidated transaction retry behavior around repeated sandbox updates followed by another `emerge --pretend` pass.

- Updated install progress calculation so active package index and completed package count are tracked separately.

- Hardened repository sync eligibility so stale inferred state cannot override the current enabled repository set.

### Known issues

- Repository and overlay management still need full real implementation.
- Package config writing still needs safer merge/replace/diff behavior.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- The keyword traversal issue still needs real-world testing across requested atoms, direct dependencies, deep dependencies, and mixed keyword/license transactions.

## [0.4.0] - 2026-09-10

### Added

- Multi-package install, rebuild, and update flows.
- `atom@version` package target syntax.
- Rebuild flow using `emerge --oneshot`.
- Update flow using `emerge --update --deep --newuse`.
- Release packaging workflow for source and vendor tarballs.

### Changed

- README now focuses on Portico’s v1 package workflow scope.
- Portico’s v1 roadmap is centered on package install, rebuild, update, and repository workflows.

### Known issues

- Repository and overlay management are not fully implemented yet.
- Transaction parsing still needs hardening.
- Config diff/merge safety still needs polish.
- USE flag picker does not yet support search/filtering.
- Bootstrap and system recipes are intentionally out of scope for v1.