# Changelog

## [0.4.0] - 2026-09-03

### Added

- Multi-package install, rebuild, and update flows.
- `atom@version` package target syntax.
- Rebuild flow using `emerge --oneshot`.
- Update flow using `emerge --update --deep --newuse`.
- Release packaging workflow for source and vendor tarballs.

### Changed

- README now focuses only on v1 package workflow scope.

### Known issues

- Repository/overlay management is not fully implemented yet.
- Transaction parsing still needs hardening.
- Config diff/merge safety still needs polish.
