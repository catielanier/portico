# Changelog

All notable changes to Portico will be documented in this file.

Portico follows semantic versioning before 1.0 loosely: patch releases may still include internal refactors when they support bug fixes.

## [0.5.1] - 2026-09-11

### Added

- Added package uninstall support.
  - New command: `sudo portico uninstall <atom...>`.
  - Supports uninstalling one or more package atoms in a single command.
  - Uses `emerge --unmerge <atom...>` for the requested packages.
  - Requires root privileges.

- Added post-uninstall depclean prompt.
  - After uninstalling the requested package or packages, Portico asks whether to run depclean.
  - Depclean only runs if the user confirms.
  - If declined, Portico exits without running depclean.

- Added explicit depclean command.
  - New command: `sudo portico clean`.
  - Runs `emerge --depclean`.
  - Requires root privileges.
  - Does not take package arguments.

- Added i18n coverage for uninstall and clean commands.
  - Added help text keys for `uninstall`.
  - Added help text keys for `clean`.
  - Added runtime message keys for uninstall, depclean prompt, depclean skipped, clean start, and clean completion.

- Added routed help coverage for uninstall and clean.
  - `portico --help uninstall`
  - `portico --help clean`

### Changed

- README now documents package removal workflows.
  - Added `uninstall` command documentation.
  - Added `clean` command documentation.
  - Clarified that uninstall uses `emerge --unmerge`.
  - Clarified that clean uses `emerge --depclean`.
  - Clarified that depclean after uninstall is prompted, not automatic.

- Root command registration now includes:
  - `uninstall`
  - `clean`

### Internal

- Added Portage cleanup wrapper logic.
  - `Unmerge()` runs `emerge --unmerge <atom...>`.
  - `Depclean()` runs `emerge --depclean`.
  - Package atom cleanup deduplicates atom arguments before passing them to Portage.

- Added CLI workflow separation for uninstall and clean.
  - `uninstall` handles targeted package removal.
  - `clean` handles standalone depclean.
  - Shared yes/no prompt helper handles the post-uninstall depclean prompt.

### Known issues

- `uninstall` currently delegates directly to `emerge --unmerge` and does not preview the removal first.
- `clean` currently delegates directly to `emerge --depclean` and does not preview the depclean first.
- `uninstall` does not remove Portico-managed package configuration entries for the removed atom.
- `clean` does not accept package arguments.
- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- Unit tests still need to be added for uninstall, clean, post-uninstall depclean prompting, and help routing.

## [0.5.0] - 2026-09-11

### Added

- Added real repository and overlay management commands.
  - `portico repo list`
  - `sudo portico repo add <name>`
  - `sudo portico repo sync`
  - `sudo portico repo sync <name>`
  - `sudo portico repo sync --preflight`
  - `sudo portico repo sync <name> --preflight`
  - `sudo portico repo remove <name>`
  - `sudo portico repo remove <name> --force`
  - `portico overlay ...` aliases for the same workflows.

- Added idempotent repository add behavior.
  - If a repository is not enabled, Portico enables it and syncs it.
  - If a repository is already enabled, Portico leaves it enabled and syncs it.
  - After enabling a repository, Portico verifies that Gentoo reports it as enabled before syncing.

- Added explicit repository sync modes.
  - `sudo portico repo sync` force-syncs all currently enabled repositories.
  - `sudo portico repo sync <name>` force-syncs one currently enabled repository.
  - `sudo portico repo sync --preflight` syncs only enabled repositories that are never-synced or stale.
  - `sudo portico repo sync <name> --preflight` syncs one repository only if it is never-synced or stale.
  - `-p` is supported as shorthand for `--preflight`.

- Added repository sync freshness tracking.
  - Portico records sync timestamps under `/var/cache/portico/repo-sync`.
  - Sync stamps are freshness metadata only.
  - Sync stamps do not define whether a repository exists or is enabled.

- Added safe repository removal behavior.
  - Portico refuses to disable the protected `gentoo` repository.
  - Before disabling a repository, Portico checks installed package metadata under `/var/db/pkg`.
  - If installed packages came from the repository, removal is blocked unless `--force` is used.
  - Forced removal prints a warning listing affected installed packages.
  - After disabling a repository, Portico verifies that it no longer appears in the enabled repository list.
  - Portico removes its own sync stamp after successful repository removal.

- Added routed help support.
  - `portico --help`
  - `portico --help find`
  - `portico --help query`
  - `portico --help install`
  - `portico --help rebuild`
  - `portico --help update`
  - `portico --help repo`
  - `portico --help repo sync`
  - `portico --help overlay remove`

- Added full help documentation keys for all current commands.
  - Root command help.
  - Package search and query help.
  - Install, rebuild, and update help.
  - Repository and overlay help.
  - Repository subcommand help for `list`, `add`, `sync`, and `remove`.

- Added version output.
  - `portico --version`
  - `portico -v`
  - Development builds default to `unstable`.
  - Release and Portage builds can inject the real version at build time.

### Changed

- Repository state is now based on Gentoo’s enabled repository state.
  - Portico treats `eselect repository list -i` as the source of truth.
  - Directories under `/var/db/repos` are treated as supporting filesystem state only.
  - Portico cache files are treated as Portico metadata only.

- Explicit repository sync and preflight sync now have separate meanings.
  - Explicit sync means “sync now.”
  - Preflight sync means “sync only if needed.”

- `repo` and `overlay` are now aliases for the same repository-management behavior.
  - `portico repo ...` and `portico overlay ...` use the same manager logic.
  - Help text is specialized so users can use either vocabulary comfortably.

- Help text is now centralized through i18n resources.
  - Command `Short`, `Long`, and `Example` fields are applied from translation keys.
  - This avoids scattering command documentation across individual CLI implementation files.

### Fixed

- Fixed removed or stale overlays being treated as sync candidates.
  - Portico no longer syncs repositories merely because a directory exists under `/var/db/repos`.
  - Portico no longer syncs repositories merely because a Portico sync stamp exists.
  - Repositories removed outside Portico do not reappear as active sync targets unless Gentoo reports them as enabled.

- Fixed `repo add` behavior for already-enabled repositories.
  - Re-running `sudo portico repo add <name>` no longer relies on `eselect` handling that case gracefully.
  - Portico checks enabled state first and syncs the repository either way.

- Fixed overly broad version flag handling.
  - `--version` and `-v` are only treated as version requests when used as the sole argument.
  - This keeps room for command-specific flags later.

### Internal

- Added repository manager support for:
  - listing enabled repositories;
  - checking whether a repository is enabled;
  - enabling and syncing repositories;
  - force-syncing one repository;
  - force-syncing all enabled repositories;
  - preflight-syncing one repository;
  - preflight-syncing all enabled repositories;
  - disabling repositories safely;
  - detecting installed packages from repository metadata;
  - deleting Portico sync stamps after removal.

- Added structured repository sync decisions.
  - `manual`
  - `never-synced`
  - `stale`
  - `not-needed`

- Added structured repository removal results.
  - Repository name.
  - Whether removal was forced.
  - Installed packages that came from the removed repository.

- Added protected repository errors.
- Added repository-in-use errors.
- Added centralized command help application for the Cobra command tree.
- Added routed help lookup for nested command paths.
- Added i18n help keys for every current Portico command.

### Known issues

- Repository removal does not migrate installed packages to another repository.
- Repository removal does not run `depclean`.
- Repository removal does not delete repository files from `/var/db/repos`.
- Repository removal does not remove package configuration entries.
- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Transaction preflight sync still needs to be wired through the new repository manager path everywhere install/rebuild/update require it.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- Unit tests still need to be added for repository list parsing, sync decisions, protected removal, in-use removal blocking, and routed help.

## [0.4.4] - 2026-09-11

### Fixed

- Fixed USE flag descriptions no longer appearing for the highlighted flag.
  - The USE picker now shows the currently highlighted flag’s description again.
  - Descriptions update immediately when moving between flags.
  - Long descriptions wrap to the available terminal width.
  - Description rendering works with paginated USE flag lists.
  - Description rendering remains valid after terminal resizing.
  - Flags without descriptions are handled with a neutral fallback.

### Changed

- Began migrating Portico-owned user-facing UI strings into the i18n layer.
  - USE picker titles, page labels, table headers, help text, button labels, and description labels now use translation keys.
  - Install progress messages now use translation keys.
  - Shared button labels such as `Confirm`, `Cancel`, `Next`, and `Prev` now use translation keys.
  - Missing translations continue to fall back safely through the existing i18n behavior.

- Added default i18n helper plumbing.
  - Added a default English translator helper for UI components that do not yet receive a translator explicitly.
  - UI components can now request translated strings without hardcoding new display copy directly.

- Improved install progress i18n coverage.
  - The compilation-time notice now uses an i18n key.
  - Progress percentage, completed package count, package count, cancel hint, and preparing state now use i18n keys.

- Updated joke wording.
  - The Arch joke now says `btw I use Arch` instead of `Actually, I use Arch`.

### Internal

- Added i18n keys for USE picker UI copy.
- Added i18n keys for install progress UI copy.
- Added common i18n keys for shared button labels.
- Added `i18n.MustDefault()` as a safe default-English translator helper.
- Updated USE picker rendering to reserve bounded description space while preserving pagination.
- Added text wrapping support for highlighted USE flag descriptions.

### Known issues

- This release starts the i18n migration but does not yet move every Portico-owned CLI string into translation resources.
- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Repository and overlay management still need full real implementation.
- Package config writing still needs safer merge/replace/diff behavior.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- The keyword traversal issue still needs real-world testing across requested atoms, direct dependencies, deep dependencies, and mixed keyword/license transactions.

## [0.4.3] - 2026-09-10

### Fixed

- Fixed dependency USE changes not being persisted before installation.
  - Dependency USE changes discovered during sandbox dependency resolution are now retained.
  - Dependency USE changes applied to the temporary Portage sandbox are now included in the final real Portage configuration write.
  - Real installs now use the same effective USE configuration that produced the successful `emerge --pretend` transaction.
  - Previously resolved dependency USE requirements should no longer reappear during the real `emerge`.

- Fixed dependency USE changes discovered during early availability checks being lost.
  - USE requirements found before the user-facing USE picker are now carried forward into the final resolved transaction state.
  - Later dependency-resolution passes merge with earlier discovered USE requirements instead of replacing them.

- Fixed dependency USE persistence for rebuild flows.
  - `portico rebuild` now carries sandbox-resolved dependency USE changes into the final real `package.use` write before running `emerge --oneshot`.

- Fixed duplicate dependency USE entries in the resolved transaction summary.
  - Repeated requirements discovered across multiple dependency-resolution passes are now deduplicated before rendering and before writing real configuration.

### Changed

- Improved transaction resolution state tracking.
  - Portico now tracks user-selected top-level USE flags and Portage-required dependency USE changes as separate parts of the resolved configuration.
  - The final configuration write now includes both categories before running the real package operation.

- Improved dependency USE rendering.
  - The transaction plan now reflects dependency USE changes that Portico actually intends to persist.
  - The displayed sandbox dependency USE changes are deduplicated for readability.

### Internal

- Added shared dependency USE deduplication helpers.
  - Dependency USE changes are keyed by atom plus normalized flag list.
  - Empty atoms and empty flag lists are ignored.
  - Duplicate dependency USE requirements are skipped.

- Updated install and rebuild resolver flows to carry existing dependency USE changes into later dependency-resolution passes.

- Updated final config application so dependency USE changes are written through the same package-scoped `package.use` path as user-selected flags.

### Known issues

- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Repository and overlay management still need full real implementation.
- Package config writing still needs safer merge/replace/diff behavior.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- The keyword traversal issue still needs real-world testing across requested atoms, direct dependencies, deep dependencies, and mixed keyword/license transactions.

## [0.4.2] - 2026-09-10

### Fixed

- Fixed license acceptance for unusual or nonstandard license identifiers.
  - Portico now preserves exact license tokens reported by Portage.
  - Uncommon or previously unseen license names are no longer rejected or normalized incorrectly.
  - License identifiers with hyphens, underscores, mixed case, or other valid punctuation are handled as reported.
  - Multiple licenses on a single package are handled.
  - Multiple packages with different license requirements are handled.
  - License requirements discovered through dependency resolution are handled iteratively.

- Fixed license parsing in masked-package reports.
  - Mask reason classification remains case-insensitive.
  - Extracted license identifiers are no longer lowercased.
  - Package-scoped license entries written to `package.license` now preserve the license text Portage actually reported.

- Fixed license parsing in autounmask reports.
  - Portico now treats the first field as the package atom and all remaining fields as exact license tokens.
  - License requirements are no longer limited to simple alphanumeric identifiers.
  - Previously unseen license identifiers can be surfaced to the user for explicit approval.

- Fixed USE flag selector input when pagination is active.
  - `↑` / `↓` and `k` / `j` now navigate visible USE flags.
  - `Space` toggles the currently selected USE flag.
  - `←` / `→` and `h` / `l` now cycle through available action buttons.
  - `Tab` and `Shift+Tab` also cycle button focus.
  - `Enter` activates the focused button.
  - `Next` advances exactly one page.
  - `Prev` moves back exactly one page.
  - Selections persist across page changes.
  - Input remains valid after terminal resizing.

### Changed

- Improved USE picker focus handling.
  - The picker now keeps flag cursor state and button focus state valid after paging or resizing.
  - Page changes preserve selection state instead of resetting or losing interaction state.
  - Button focus is revalidated whenever the available actions change.
  - The USE picker now uses Bubble Tea’s alt-screen mode for cleaner terminal rendering.

- Improved license handling philosophy.
  - Portico now treats Portage’s reported license identifiers as authoritative.
  - Portico still requires explicit user confirmation before accepting licenses.
  - Portico still writes package-scoped license entries instead of modifying global `ACCEPT_LICENSE`.

### Internal

- Hardened autounmask parsing for:
  - USE changes
  - keyword changes
  - license changes

- Hardened masked-package parsing so detection can normalize text for classification without mutating extracted values.

- Updated USE picker state mutation to use pointer receivers where state must persist.

- Added explicit validation for focused action buttons after page changes and terminal resize events.

### Known issues

- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Repository and overlay management still need full real implementation.
- Package config writing still needs safer merge/replace/diff behavior.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- The keyword traversal issue still needs real-world testing across requested atoms, direct dependencies, deep dependencies, and mixed keyword/license transactions.

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