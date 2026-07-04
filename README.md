# Portico

A clearer CLI/TUI entrance to Gentoo Portage for choosing package features and safely running `emerge`.

Portico is a guided terminal tool for Gentoo users who want to understand package options before installing, rebuilding, or updating software. It helps inspect packages, review USE flags, write per-package configuration, preview Portage transactions, and then hand execution back to Portage.

Portico does **not** replace Portage.

Portico explains and stages changes. Portage remains the source of truth.

---

## Goals

Portico exists to make Gentoo's power easier to approach without hiding how Gentoo works.

Portico should:

- explain package options before installation
- make USE flags easier to inspect and select
- prefer per-package configuration over global changes
- show exactly what will be written before writing files
- show exactly what commands will run before running them
- preserve raw Portage output for troubleshooting
- keep read-only commands usable without root
- require confirmation before mutating the system
- respect Gentoo's flexibility instead of pretending it does not exist

Portico should not:

- replace `emerge`
- replace the Gentoo Handbook
- silently mutate global system policy
- hide Portage warnings
- silently overwrite configuration
- globally enable unstable keywords
- globally accept licenses
- automate untested bootloader workflows
- pretend source-based systems are magic

---

## Planned Commands

```sh
portico query <atom>
sudo portico install <atom>
sudo portico rebuild <atom>
sudo portico update
sudo portico update <atom>
```

Examples:

```sh
portico query media-video/obs-studio
sudo portico install media-video/obs-studio
sudo portico rebuild media-video/ffmpeg
sudo portico update
sudo portico update www-client/firefox
```

---

## Core Idea

A normal Portico package flow should look like this:

```text
portico install <atom>

-> inspect package metadata
-> show available USE flags
-> explain USE flag descriptions
-> let the user select desired functionality
-> generate per-package package.use entries
-> show a reviewable plan
-> ask for confirmation
-> write Portico-managed config
-> run emerge --pretend --verbose
-> show Portage's transaction preview
-> ask again
-> run emerge --ask --verbose
```

Portico should make the workflow clearer, but it should not bypass Portage's own resolver, warnings, or confirmation model.

---

## Safety Model

Before making changes, Portico should show a plan.

Example:

```text
Portico Plan

Action:
  Install media-video/ffmpeg

Portico will:

  ✓ write a per-package USE entry
    /etc/portage/package.use/portico

  ✓ enable:
    opus x264 x265 vaapi pipewire

  ✓ disable:
    pulseaudio jack

  ✓ run emerge --pretend --verbose media-video/ffmpeg
  ✓ ask before running emerge --ask --verbose media-video/ffmpeg

Portico will not:

  ✗ modify global USE flags in /etc/portage/make.conf
  ✗ overwrite user-managed package.use entries
  ✗ configure unrelated media packages
  ✗ teach ffmpeg to pronounce "GIF" correctly

Continue? [y/N]
```

The "will not" section is part of Portico's safety contract. It should include serious boundaries and may end with one harmless scoped joke.

---

## Configuration Philosophy

Portico should prefer scoped, reviewable configuration.

For v0.x, package USE changes should be written to:

```text
/etc/portage/package.use/portico
```

Example:

```text
media-video/ffmpeg opus x264 x265 vaapi pipewire -pulseaudio -jack
```

Portico should not silently edit:

```text
/etc/portage/make.conf
```

Global policy changes should always be explicit.

---

## Kernel-Aware Installs

Kernel packages are a special case because installing or rebuilding a kernel may require bootloader follow-up.

Portico should detect kernel-related packages and guide the user through the next step instead of pretending kernels are ordinary packages.

Initial v0.1 support should focus on GRUB, because it is the only bootloader path currently planned for direct testing.

Example:

```text
Portico detected a kernel package.

Detected bootloader:
  GRUB

Portico can regenerate GRUB configuration after the kernel install:

  grub-mkconfig -o /boot/grub/grub.cfg

Include this step? [Y/n]
```

Portico v0.1 should not:

- install GRUB
- modify EFI boot entries
- configure Secure Boot
- automate untested bootloader workflows
- assume a `*-sources` package produced a bootable kernel

Example plan item:

```text
Portico will not:

  ✗ install GRUB to your EFI system partition
  ✗ modify EFI boot entries
  ✗ configure Secure Boot
  ✗ automate untested bootloader workflows
  ✗ become your BIOS therapist
```

---

## Bootloader Support

Portico should only automate bootloader workflows that maintainers or contributors can actually test.

Planned v0.x support:

- [x] GRUB planned for v0.1
- [ ] systemd-boot detection
- [ ] rEFInd detection
- [ ] EFISTUB/direct UEFI detection
- [ ] LILO/ELILO detection
- [ ] Syslinux/Extlinux detection
- [ ] Limine detection

Non-GRUB bootloaders may be detected and reported before they are automated.

Portico should not automate untested bootloader workflows.

---

## Dependencies

Portico is written in Go.

Planned Go libraries:

- [`bubbletea`](https://github.com/charmbracelet/bubbletea) — TUI runtime
- [`bubbles`](https://github.com/charmbracelet/bubbles) — reusable TUI components
- [`lipgloss`](https://github.com/charmbracelet/lipgloss) — terminal styling
- [`cobra`](https://github.com/spf13/cobra) — CLI commands
- [`go-i18n`](https://github.com/nicksnyder/go-i18n) — localization
- [`toml`](https://github.com/BurntSushi/toml) — configuration and joke packs

Gentoo-side tools likely used during development:

- `emerge`
- `equery` from `app-portage/gentoolkit`
- `portageq`
- Portage configuration files under `/etc/portage`

Portico should treat Portage as the source of truth.

---

## Development

Clone the repository:

```sh
git clone https://github.com/YOUR_GITHUB_USERNAME/portico.git
cd portico
```

Download module dependencies:

```sh
go mod tidy
```

Run tests:

```sh
go test ./...
```

Run locally:

```sh
go run ./cmd/portico --help
```

Build:

```sh
go build -o portico ./cmd/portico
```

Run the built binary:

```sh
./portico --help
```

Install into your Go bin path:

```sh
go install ./cmd/portico
```

Make sure your Go bin directory is in your `PATH`:

```sh
export PATH="$HOME/go/bin:$PATH"
```

---

## Project Structure

Planned structure:

```text
portico/
  cmd/
    portico/
      main.go
  internal/
    bootloader/
    cli/
    i18n/
    jokes/
    plan/
    portage/
    ui/
  README.md
  LICENSE
  go.mod
  go.sum
```

Suggested package responsibilities:

```text
cmd/portico
  Program entrypoint.

internal/cli
  Cobra commands and command routing.

internal/portage
  Portage, emerge, equery, and package metadata interaction.

internal/plan
  Reviewable action plans.

internal/ui
  Terminal rendering and Bubble Tea models.

internal/i18n
  Translation setup and message lookup.

internal/jokes
  Scoped joke selection and joke pack loading.

internal/bootloader
  Kernel/bootloader detection and post-kernel install planning.
```

---

# Roadmap

Portico is planned in layers. Each layer should remain explicit, reviewable, and Portage-respecting.

Portico should explain and stage changes; Portage remains the source of truth.

---

## v0.x — Portage Workflow Assistant

The v0.x series focuses on making everyday package operations easier to understand and safer to perform.

Portico should help users inspect packages, choose USE flags, write per-package configuration, preview emerge operations, and install or rebuild packages with clear confirmation.

### Core Commands

```sh
portico query <atom>
sudo portico install <atom>
sudo portico rebuild <atom>
sudo portico update
sudo portico update <atom>
```

### v0.1 — Core Package Workflow

- [ ] Initialize CLI command structure
- [ ] Add `portico query <atom>`
- [ ] Add `portico install <atom>`
- [ ] Add `portico rebuild <atom>`
- [ ] Query package metadata
- [ ] Show package description
- [ ] Show available USE flags
- [ ] Show USE flag descriptions
- [ ] Show current enabled/disabled USE state
- [ ] Interactively select desired USE flags
- [ ] Generate per-package `package.use` entries
- [ ] Confirm generated changes before writing
- [ ] Write Portico-managed entries to `/etc/portage/package.use/portico`
- [ ] Run `emerge --pretend --verbose`
- [ ] Show the Portage preview before installation
- [ ] Run `emerge --ask --verbose` only after confirmation
- [ ] Preserve raw Portage/equery output for troubleshooting

### v0.1 — Safety Model

Before making changes, Portico should show:

- [ ] selected package atom
- [ ] selected USE flags
- [ ] generated `package.use` entries
- [ ] files that will be written or modified
- [ ] commands that will be run
- [ ] what Portico will do
- [ ] what Portico will not do
- [ ] one final scoped joke item at the bottom of the "will not" list

Portico should not:

- [ ] silently modify global `USE` flags
- [ ] silently overwrite user-managed files
- [ ] hide Portage output
- [ ] install, rebuild, update, or remove packages without confirmation
- [ ] resolve dependencies independently of Portage

### v0.1 — Kernel-Aware Installs

Kernel packages are a special case because installing or rebuilding a kernel may require bootloader follow-up.

Initial support:

- [ ] Detect kernel-related packages
- [ ] Distinguish kernel source packages from package-managed kernel packages where possible
- [ ] Detect GRUB when available
- [ ] Offer to regenerate GRUB configuration after successful kernel installation
- [ ] Show the exact GRUB command before running it

Example:

```sh
grub-mkconfig -o /boot/grub/grub.cfg
```

Portico v0.1 should not:

- [ ] install GRUB
- [ ] modify EFI boot entries
- [ ] configure Secure Boot
- [ ] automate untested bootloader workflows
- [ ] assume a `*-sources` package produced a bootable kernel

### v0.2 — Safer Editing

- [ ] Read existing Portico-managed `package.use` entries
- [ ] Preserve manual `package.use` entries
- [ ] Show diffs before writing files
- [ ] Backup files before modification
- [ ] Improve package atom validation
- [ ] Improve error handling when Portage rejects a configuration
- [ ] Return to USE flag selection after failed pretend emerge
- [ ] Detect when selected USE flags require a rebuild

### v0.3 — Portage-Aware UX

- [ ] Detect masked USE flags
- [ ] Detect forced USE flags
- [ ] Surface `REQUIRED_USE` failures clearly
- [ ] Show likely causes of Portage conflicts
- [ ] Group USE flags by rough feature category
- [ ] Search/filter USE flags in the picker
- [ ] Show installed version and available version
- [ ] Show repository/source where available
- [ ] Show raw Portage output on request

### v0.4 — Update Workflow

- [ ] Add `sudo portico update`
- [ ] Add `sudo portico update <atom>`
- [ ] Preview world updates
- [ ] Preview single-package update transactions
- [ ] Show packages to install, update, rebuild, downgrade, or remove
- [ ] Show dependency updates required by Portage
- [ ] Allow accept/cancel for update plans
- [ ] Allow target selection for world updates where safe
- [ ] Re-run pretend emerge after target selection changes
- [ ] Clearly explain that Portico selects targets and Portage resolves transactions
- [ ] Never run `depclean` automatically

### Pre-1.0 — Package Visibility Helpers

- [ ] Detect keyword-masked packages
- [ ] Detect license blocks
- [ ] Detect hard-masked packages
- [ ] Detect missing repository/overlay availability
- [ ] Add scoped unmasking helper
- [ ] Add scoped license acceptance helper
- [ ] Show exact files and entries before writing
- [ ] Never globally enable unstable keywords
- [ ] Never globally accept licenses
- [ ] Treat hard-masked packages as high-risk and require explicit confirmation

Possible commands:

```sh
sudo portico unmask <atom>
sudo portico accept-license <atom>
```

### Pre-1.0 — Overlay Awareness

Portico should understand packages from enabled repositories and overlays before 1.0.

- [ ] Detect when a package exists in multiple repositories
- [ ] Show available versions by repository
- [ ] Show whether a package is stable, testing, masked, or unavailable under the current profile
- [ ] Let the user choose which repository/source to install from
- [ ] Explain when accepting keywords, unmasking, or repository-specific configuration is required
- [ ] Generate reviewable changes before writing files
- [ ] Run `emerge --pretend --verbose` before installation

Portico should not:

- [ ] silently enable overlays
- [ ] globally prefer overlays over the official Gentoo repository
- [ ] globally unmask packages
- [ ] globally change accepted keywords

### Pre-1.0 — Localization

Portico v0.x may initially ship English-only output, but the codebase should be localization-ready from the beginning.

- [ ] Centralize user-facing strings
- [ ] Use translation keys for UI labels, prompts, warnings, errors, and plan items
- [ ] Support locale-aware joke packs
- [ ] Allow jokes to be extended, replaced, or disabled
- [ ] Display Gentoo/Portage package descriptions and USE flag descriptions as provided by the system
- [ ] Prepare contributor documentation for translations

### Pre-1.0 — Bootloader Awareness

GRUB is the initial supported bootloader path.

Additional bootloader support may be added if maintainers or contributors can test the workflow.

- [x] GRUB planned for v0.1
- [ ] systemd-boot detection
- [ ] rEFInd detection
- [ ] EFISTUB/direct UEFI detection
- [ ] LILO/ELILO detection
- [ ] Syslinux/Extlinux detection
- [ ] Limine detection
- [ ] Bootloader-specific post-kernel instructions
- [ ] Bootloader-specific plan items after explicit confirmation

Portico should not automate untested bootloader workflows.

---

## v1.x — System Configuration Assistant

The v1.x series expands Portico beyond package installation into optional post-install configuration.

This layer should help installed packages become usable while keeping every file write and service change explicit.

Possible commands:

```sh
sudo portico configure <target>
sudo portico recipe <name>
sudo portico service enable <service>
```

### v1.0 — Config File Recipes

- [ ] Add optional post-install recipes
- [ ] Show recipe plans before applying
- [ ] Write starter configuration files only after confirmation
- [ ] Show diffs before overwriting files
- [ ] Backup existing files before modification
- [ ] Clearly mark Portico-managed files
- [ ] Allow users to decline specific recipe steps
- [ ] List declined actions in the "will not" section

Example:

```text
Portico will not:
  ✗ create ~/.xinitrc
    You declined graphical autostart setup.
```

### v1.x — Graphical Session Setup

- [ ] Xorg starter recipes
- [ ] Wayland starter recipes
- [ ] Openbox starter recipe
- [ ] Sway starter recipe
- [ ] Display manager setup recipes
- [ ] Manual `startx` / compositor launch recipes
- [ ] `.xinitrc` generation where appropriate
- [ ] Session entry generation where appropriate
- [ ] Detect and explain when graphical autostart is not configured

Portico should not:

- [ ] choose a desktop environment without asking
- [ ] enable a display manager without confirmation
- [ ] overwrite existing graphical configuration silently

### v1.x — Audio Setup

- [ ] PipeWire starter recipe
- [ ] WirePlumber starter recipe
- [ ] PulseAudio compatibility guidance
- [ ] JACK-related guidance where appropriate
- [ ] Audio-production-oriented presets
- [ ] Show service/session startup requirements
- [ ] Avoid disabling existing audio configs without explicit confirmation

### v1.x — Service Manager Awareness

Portico should become aware of service managers for recipes and service enablement.

Possible targets:

- [ ] OpenRC
- [ ] runit
- [ ] systemd
- [ ] s6/s6-rc, if contributors appear

Portico should:

- [ ] detect current service manager
- [ ] show service enablement plans
- [ ] support service-manager-specific recipes
- [ ] avoid service changes outside the stated plan
- [ ] avoid init-system migration unless explicitly requested in a future feature

### v1.x — Presets

- [ ] Desktop workstation preset
- [ ] Gaming preset
- [ ] Audio-production preset
- [ ] Development workstation preset
- [ ] Minimal graphical preset
- [ ] Explain what each preset enables before applying
- [ ] Allow users to review and deselect preset components

### v1.x — Experimental Init/Service Migration Planning

This is not core v1 functionality, but may be explored if contributors with deep system knowledge are available.

Possible future command:

```sh
sudo portico init migrate <target>
```

Initial scope should be plan-first:

- [ ] Detect current init/service manager
- [ ] Detect enabled services
- [ ] Map known services to target service manager
- [ ] Mark unknown services for manual review
- [ ] Show migration plan before applying
- [ ] Avoid removing the previous init system automatically
- [ ] Avoid changing bootloader init targets without explicit confirmation

Portico should not blindly migrate init systems.

---

## v2.x — Bootstrap Assistant

The v2.x series is the final-boss feature set.

This layer would allow Portico to guide a user from a new Gentoo chroot/base install to a ready-to-run system.

The intended command:

```sh
portico bootstrap
```

This command is intended to be run as root from a Gentoo chroot during or after a base install.

### v2.0 — Bootstrap Flow

Portico should guide the user through system setup choices, generate a reviewable setup plan, and apply that plan only after explicit confirmation.

Possible flow:

```sh
emerge --sync
emerge app-portage/portico
portico bootstrap
```

### v2.x — Bootstrap Scope

Possible setup areas:

- [ ] system role selection
- [ ] hostname
- [ ] locale
- [ ] timezone
- [ ] user accounts
- [ ] kernel strategy
- [ ] firmware
- [ ] init/service manager
- [ ] bootloader setup
- [ ] package sets
- [ ] USE flag configuration
- [ ] graphical session setup
- [ ] audio stack setup
- [ ] networking setup
- [ ] service enablement
- [ ] starter configuration files

### v2.x — Bootstrap Planning

Before applying changes, Portico should show:

- [ ] selected system role
- [ ] selected init system
- [ ] selected kernel strategy
- [ ] selected bootloader strategy
- [ ] selected desktop/session
- [ ] selected audio stack
- [ ] packages to install
- [ ] USE flags to write
- [ ] files to write or modify
- [ ] services to enable
- [ ] commands to run
- [ ] actions the user declined
- [ ] actions Portico will not perform

### v2.x — Bootstrap Boundaries

Portico bootstrap should not:

- [ ] replace the Gentoo Handbook
- [ ] partition disks
- [ ] format filesystems
- [ ] assume one correct desktop, init system, bootloader, or audio stack
- [ ] hide generated configuration
- [ ] apply changes without a review step
- [ ] modify EFI boot entries without explicit confirmation
- [ ] configure Secure Boot without explicit support
- [ ] pretend Gentoo is not Gentoo

### v2.x — Official Repository Goal

For `portico bootstrap` to be realistic as a recommended new-user path, Portico should ideally be available from the official Gentoo repository.

Target package:

```text
app-portage/portico
```

Before v2 is considered realistic, Portico should be:

- [ ] packaged as an ebuild
- [ ] usable from the official Gentoo tree or an accepted Gentoo-maintained path
- [ ] tested from a fresh Gentoo chroot
- [ ] documented for post-stage3 setup
- [ ] conservative enough for documentation to recommend as an optional guided workflow

---

## Joke System

Portico may include one scoped joke line at the bottom of the "will not" section.

The joke system should be:

- optional
- configurable
- localizable
- harmless
- scoped to context when possible

Example:

```text
Portico will not:

  ✗ modify global USE flags in /etc/portage/make.conf
  ✗ overwrite user-managed package.use entries
  ✗ tell you to "just use Rust"
```

Possible configuration modes:

```toml
[jokes]
mode = "extend"
```

Supported modes should eventually include:

```text
extend
replace
off
```

Joke packs may live in locations such as:

```text
/etc/portico/jokes.d/*.toml
~/.config/portico/jokes.toml
```

---

## Localization

Portico may begin as English-only, but it should be localization-ready from the beginning.

User-facing strings should be centralized and referenced by translation keys.

Portico-owned UI should be localizable:

- labels
- prompts
- warnings
- errors
- plan items
- confirmation text
- help output
- joke packs

Gentoo-provided metadata, package descriptions, and USE flag descriptions should be displayed as provided by the system.

---

## License

Portico is licensed under the GNU General Public License v3.0 or later.

See [`LICENSE`](./LICENSE).
