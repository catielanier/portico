# Changelog

All notable changes to Portico will be documented in this file.

Portico follows semantic versioning before 1.0 loosely: patch releases may still include internal refactors when they support bug fixes.

## 0.5.5

### Added

- Added live `REQUIRED_USE` validation to the interactive USE flag picker.
  - Portico now checks the selected ebuild's `REQUIRED_USE` constraints while USE flags are being configured.
  - The Confirm action is disabled while the current selection violates a `REQUIRED_USE` condition.
  - Violations are displayed directly in the picker instead of allowing a known-invalid configuration to continue.
  - Supports nested conditions and the common `REQUIRED_USE` operators:
    - `|| ( ... )` — at least one
    - `^^ ( ... )` — exactly one
    - `?? ( ... )` — at most one
    - conditional expressions such as `foo? ( bar !baz )`
  - Existing `+` / `-` state indicators remain authoritative; color and warnings supplement rather than replace them.

- Added a second `REQUIRED_USE` safety check during sandbox dependency resolution.
  - `emerge --pretend` remains the authoritative transaction validator.
  - If Portage reports a `REQUIRED_USE` failure that escaped live validation, Portico intercepts it instead of treating it as a generic pretend failure.
  - Direct-package conflicts can return to USE configuration and retry through the temporary sandbox.
  - Real Portage configuration is not modified until the resulting transaction is valid and confirmed.

- Added centralized, theme-aware terminal color roles.
  - Success indicators use terminal-theme green.
  - Errors and blocked actions use terminal-theme red.
  - Warnings use terminal-theme yellow.
  - Informational highlights use terminal-theme cyan.
  - Interactive focus uses a terminal-native accent style.
  - Muted and disabled UI uses terminal-native dim styling.
  - No RGB or hex palette is required for the default UI.

- Added semantic colorization throughout Portico, including:
  - plans and status indicators
  - USE flag selection
  - `REQUIRED_USE` warnings
  - package source selection
  - repository operations
  - dependency and sandbox steps
  - install and cleanup progress
  - keyword, license, and mask handling
  - query and search output

- Added no-color handling.
  - Respects `NO_COLOR`.
  - Disables Portico-owned color with `CLICOLOR=0`.
  - Degrades cleanly for `TERM=dumb`.
  - Avoids ANSI styling when output is not attached to a terminal.

- Added CI testing for pull requests.
  - Pull requests now run the complete Go test suite with `go test ./...`.

- Added automated Gentoo dependency archive generation for tagged releases.
  - Version tags now produce:
    - `portico-<version>.tar.gz`
    - `portico-<version>-deps.tar.xz`
  - The dependency archive uses a populated Go module cache suitable for the Portico Gentoo ebuild.

### Changed

- The USE picker now prevents confirmation of configurations Portico already knows violate the selected package's `REQUIRED_USE` rules.

- Package metadata resolution for `REQUIRED_USE` now resolves the target package version through Portage before reading the ebuild metadata.

- Progress-bar styling now uses Portico's terminal-native semantic palette instead of the default hardcoded RGB colors provided by the underlying UI component.

- Interactive focus styling is now consistent across Portico while retaining non-color focus indicators for accessibility.

## [0.5.4] - 2026-09-11

### Added

- Added package source selection for install targets available from multiple enabled repositories.
  - Portico now checks whether an unqualified install atom is available from more than one repository or overlay.
  - If multiple repositories provide the requested atom, Portico opens a source picker before continuing.
  - The source picker shows:
    - repository / overlay name
    - latest available version from that repository
    - source status
    - explicit install target
  - The picker is scrollable for packages with many source candidates.

- Added package source discovery using `emerge -pvO <atom>`.
  - Portico parses source candidates from Portage pretend output.
  - Masked package candidates are included when Portage reports them.
  - Candidate output is grouped by repository.
  - Portico keeps the latest candidate per repository based on Portage’s output order.

- Added explicit repository selection syntax for install arguments.
  - Supports `atom::repository`.
  - Supports `atom@version::repository`.
  - Explicit repository targets skip the package source picker.
  - `atom@version::repository` resolves to a Portage-compatible target like:

    ```text
    =category/package-version::repository
    ```

- Added source picker i18n strings.
  - Multiple-source notice.
  - Source selection title.
  - Empty source message.
  - Source picker header.
  - Scroll position.
  - Selected target display.
  - Masked candidate display.
  - Source picker help text.
  - Source selection cancellation message.

### Changed

- Install command help now documents repository-qualified install targets.
  - Usage now supports:

    ```text
    install <atom[@version][::repository]...>
    ```

  - Help examples now include:
    - `sudo portico install mail-client/mailspring-bin::guru`
    - `sudo portico install mail-client/mailspring-bin@1.23.0::guru`

- README now documents package source selection.
  - Added install target forms:
    - `atom`
    - `atom@version`
    - `atom::repository`
    - `atom@version::repository`
  - Added package source picker behavior.
  - Added example source picker output.
  - Clarified that explicit `::repository` targets bypass the source picker.
  - Clarified that Portico will not silently choose between multiple enabled repositories when source selection is needed.

- Install flow now resolves package sources before USE flag inspection.
  - Requested install atoms are parsed first.
  - Repository sync runs before source discovery.
  - Unqualified multi-source atoms are resolved through the picker.
  - Selected source targets continue through the existing USE flag, sandbox, pretend, mask, keyword, license, confirmation, and install workflow.

- Source candidate ordering now preserves Portage output order.
  - Portico trusts `emerge -pvO` ordering for version priority.
  - Only the first candidate per repository is kept.
  - The `gentoo` repository may be nudged first when present, but overlays otherwise keep Portage’s reported order.

### Fixed

- Fixed package source discovery initially using the wrong discovery approach.
  - Replaced the earlier `equery list -p <atom>` idea with `emerge -pvO <atom>`, which exposes the useful masked candidate shape needed for overlay/source selection.

- Fixed source candidate tests expecting Portage output order while implementation sorted masked overlays alphabetically.
  - Candidate grouping now preserves source order from Portage output.

- Fixed non-constant format string usage in source selection cancellation.
  - Replaced direct `fmt.Errorf(translator.T(...))` with a safe constant format string.

- Fixed masked package keyword fallback behavior in install handling.
  - Portico no longer falls back to `~amd64` when it cannot parse a required keyword.
  - If a keyword mask is detected but the required keyword token cannot be determined, Portico stops instead of inventing an architecture-specific keyword.

### Internal

- Added package source discovery layer.
  - New `PackageSourceCandidate` type.
  - New `PackageSourceReport` type.
  - New `FindPackageSourceCandidates()` function.
  - New `ParsePackageSourceReport()` parser.
  - New `NeedsPackageSourceSelection()` helper.

- Added source picker UI.
  - New scrollable Bubble Tea picker for package source candidates.
  - Supports keyboard navigation:
    - `↑` / `↓`
    - `k` / `j`
    - `PgUp` / `PgDn`
    - `Home` / `End`
    - `Enter`
    - `Esc` / `q`

- Added CLI source-resolution helper.
  - Resolves install atoms before the existing install workflow.
  - Bypasses source selection when an atom already contains an explicit `::repository` qualifier.
  - Replaces ambiguous atoms with explicit selected install targets.

- Added parser tests for source discovery.
  - Covered masked package output from `emerge -pvO`.
  - Covered latest-per-repository behavior.
  - Covered multi-repository source selection detection.
  - Covered single-repository no-picker behavior.

### Known issues

- Source picker currently applies to `install`; `query`, `rebuild`, and `update <atom>` do not yet use source selection.
- Source discovery depends on Portage output shape from `emerge -pvO`.
- Version ordering trusts Portage output instead of implementing Gentoo version comparison internally.
- Source picker displays basic status only; masked/keyword/license details can be improved.
- Explicit repository syntax still depends on install target parsing accepting `::repository` correctly.
- USE flag inspection may need additional normalization if `equery` does not accept fully explicit `=category/package-version::repository` targets in all cases.
- `uninstall` does not remove Portico-managed package configuration entries for removed atoms.
- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- Unit tests still need to be added for full source picker UI behavior and explicit `atom@version::repository` argument normalization.

## [0.5.3] - 2026-09-11

### Fixed

- Fixed architecture keyword handling for masked package resolution.
  - Portico no longer only recognizes keyword masks that start with `~`.
  - Stable architecture keywords such as `arm64`, `riscv`, `ppc64`, and `x86` can now be extracted from Portage mask output.
  - Testing architecture keywords such as `~amd64`, `~arm64`, `~riscv`, and `~x86` continue to be preserved exactly.
  - Missing keyword masks remain unsupported instead of being guessed around.

- Fixed masked package keyword parsing being implicitly amd64-shaped.
  - Portico now treats required keyword tokens as opaque Portage-provided values.
  - Portico preserves the exact keyword token reported by Portage when writing package-specific `package.accept_keywords` entries.

### Internal

- Added regression tests for non-amd64 keyword handling.
  - Autounmask keyword parsing now has coverage for `~arm64`, `riscv`, `ppc64`, and `~x86`.
  - Masked package parsing now has coverage for both testing and stable non-amd64 keyword masks.
  - Accept-keyword writing now has coverage to ensure keyword tokens are written exactly as provided.

### Known issues

- Cross-architecture behavior still depends on Portage output shape and needs broader real-system testing.
- `uninstall` does not remove Portico-managed package configuration entries for removed atoms.
- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Transaction parsing still needs hardening for more Portage output shapes.

## [0.5.2] - 2026-09-11

### Added

- Added cleanup progress UI for package removal workflows.
  - New cleanup progress view for uninstall and depclean operations.
  - Shows active package removal progress when Portage reports package counts.
  - Shows completed package count and percentage when totals are available.
  - Keeps the UI active while cleanup operations are running.

- Added cancellable cleanup execution.
  - Cleanup progress can be cancelled with `Ctrl+C`, `Esc`, `q`, or `c`.
  - Cancellation propagates to the running Portage process through context cancellation.
  - Portico now shows a cleanup-cancelled message when cancellation is requested.

- Added Portage countdown detection for removal operations.
  - Portico detects Portage’s pre-removal waiting/countdown output.
  - The cleanup UI displays a waiting state before removal begins.
  - The waiting state makes the final cancellation window visible in Portico’s UI instead of leaving it hidden in raw Portage output.

- Added preview-and-confirm flow for uninstall.
  - `sudo portico uninstall <atom...>` now previews `emerge --pretend --unmerge <atom...>`.
  - Portico asks for confirmation before running `emerge --unmerge`.
  - The actual unmerge runs through the cleanup progress UI.

- Added preview-and-confirm flow for clean.
  - `sudo portico clean` now previews `emerge --pretend --depclean`.
  - Portico asks for confirmation before running `emerge --depclean`.
  - The actual depclean runs through the cleanup progress UI.

### Changed

- Post-uninstall depclean now uses the full clean workflow.
  - After uninstall finishes, Portico asks whether to start the clean workflow.
  - If confirmed, Portico previews depclean first.
  - Portico then asks for depclean confirmation before running it.
  - Post-uninstall depclean no longer jumps directly into `emerge --depclean`.

- Uninstall and clean now follow the same Portico interaction pattern as package installation.
  - Preview the Portage operation.
  - Ask for confirmation.
  - Run the real operation with a progress UI.
  - Surface cancellation clearly.

- Clean command behavior is now explicit.
  - `sudo portico clean` is a Portico-managed wrapper around `emerge --depclean`.
  - It does not take package arguments.

### Internal

- Added `internal/ui/cleanprogress.go`.
  - Provides `RunCleanupProgress`.
  - Provides `CleanupProgressEvent`.
  - Handles progress display, spinner updates, waiting/countdown state, completion, and cancellation.

- Added cleanup support in the Portage layer.
  - `PretendUnmerge()` runs `emerge --pretend --unmerge <atom...>`.
  - `PretendDepclean()` runs `emerge --pretend --depclean`.
  - `UnmergeContext()` runs `emerge --unmerge <atom...>` with progress callbacks.
  - `DepcleanContext()` runs `emerge --depclean` with progress callbacks.

- Added cleanup output parsing.
  - Parses `>>> Unmerging (N of M)` progress lines.
  - Detects Portage waiting/countdown output.
  - Parses cleanup totals from pretend output when available.
  - Keeps a short tail of Portage output for failure messages.

- Added i18n keys for cleanup progress UI.
  - Cleanup preparing state.
  - Package count.
  - Progress percentage.
  - Completed count.
  - Cancel hint.
  - Waiting/countdown state.
  - Cleanup cancelled state.

### Fixed

- Fixed post-uninstall depclean flow bypassing preview.
  - Depclean after uninstall now behaves exactly like running `sudo portico clean`.

- Fixed uninstall/clean UX being too raw compared with install.
  - Removal operations now have Portico-owned preview, confirmation, progress, and cancellation behavior.

### Known issues

- Cleanup progress depends on Portage output shape and may show limited progress if Portage does not emit parseable package counts.
- `uninstall` does not remove Portico-managed package configuration entries for removed atoms.
- `clean` does not accept package arguments.
- Safe config upsert/merge is still needed so Portico does not append duplicate or conflicting entries to its own `90-portico` files.
- Transaction parsing still needs hardening for more Portage output shapes.
- Update flow can apply license changes, but still does not apply update-time USE autounmask changes.
- USE flag picker does not yet support search/filtering.
- Unit tests still need to be added for uninstall, clean, cleanup progress parsing, countdown detection, cancellation behavior, and routed help.

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
