# Portico

A clearer CLI/TUI entrance to Gentoo Portage for choosing package features and safely running `emerge`.

Portico does not replace Portage. It wraps common package workflows with a reviewable plan, scoped config writes, and a friendlier USE flag flow.

## What Portico does

Portico helps with:

- searching for packages
- inspecting package USE flags
- installing one or more packages
- rebuilding one or more packages with revised USE flags
- updating packages
- managing repositories / overlays

Portico favors package-specific configuration and explicit confirmation before making system changes.

## What Portico will not do

Portico will not:

- replace the Gentoo Handbook
- hide Portage warnings
- silently modify global USE flags
- globally enable unstable keywords
- globally accept licenses
- overwrite user-managed Portage config without showing what it intends to do
- run package mutations without root privileges
- treat donated mirrors like a stress toy

## Commands

### Search for packages

```sh
portico find <query>
```

Searches package names and descriptions.

Example:

```sh
portico find irssi
```

### Inspect a package

```sh
portico query <atom>
```

Shows package USE flags and descriptions.

Example:

```sh
portico query net-irc/irssi
```

### Install packages

```sh
sudo portico install <atom...>
```

Configures selected USE flags, previews the Portage transaction, and installs the requested package or packages.

Example:

```sh
sudo portico install net-irc/irssi app-misc/tmux
```

Portico will:

- inspect USE flags for each requested package
- let you choose package-specific USE flag changes
- create a temporary Portage config sandbox
- run `emerge --pretend --verbose`
- resolve supported package-specific keyword, license, and dependency USE requirements
- show the calculated transaction
- ask before writing real config or running `emerge`
- run one combined `emerge` transaction

Portico will not:

- modify `/etc/portage/make.conf`
- globally enable testing keywords
- globally accept licenses
- silently run `emerge --ask`

### Rebuild packages

```sh
sudo portico rebuild <atom...>
```

Revises USE flags and rebuilds one or more packages with `emerge --oneshot`.

Example:

```sh
sudo portico rebuild net-irc/irssi
```

Portico will:

- inspect current USE flags
- let you revise package-specific USE choices
- preview the rebuild transaction
- write scoped Portage config
- run `emerge --oneshot`

Portico will not:

- add rebuilt packages to `@world`
- uninstall packages first
- run `depclean`
- globally change USE flags

### Update packages

```sh
sudo portico update
```

Updates the world set.

```sh
sudo portico update <atom...>
```

Updates one or more specific packages.

Portico previews the transaction before running the update.

### Manage repositories

```sh
portico repo list
sudo portico repo add <name>
sudo portico repo remove <name>
sudo portico repo sync <name>
```

`overlay` is also available as an alias:

```sh
portico overlay list
sudo portico overlay add <name>
sudo portico overlay remove <name>
sudo portico overlay sync <name>
```

Portico uses `repo` as the canonical command and `overlay` because Gentoo users say overlay.

## Safety model

Portico writes its own scoped config files under `/etc/portage`, such as:

```text
/etc/portage/package.use/90-portico
/etc/portage/package.accept_keywords/90-portico
/etc/portage/package.license/90-portico
```

When a package requires `~amd64`, Portico writes a package-specific keyword entry:

```text
media-video/obs-studio ~amd64
```

It does not write:

```text
ACCEPT_KEYWORDS="~amd64"
```

When Portage requires dependency USE changes, Portico writes scoped `package.use` entries for the affected packages.

Unsupported mask types stop the transaction instead of being guessed around.

## Repository syncing

Portico avoids unnecessary repository syncing.

Read-only commands such as `find` and `query` do not normally sync repositories. If Portico detects an enabled repository that has never been synced, it warns and asks whether to sync it.

Mutation commands such as `install`, `rebuild`, and `update` sync when needed.

## Requirements

Portico expects a Gentoo system with Portage available.

Recommended tools:

```sh
emerge app-portage/gentoolkit
```

Portico uses Gentoo tools such as:

- `emerge`
- `equery`
- `emaint`
- `eselect repository`

## Development

Build:

```sh
go build -o portico ./cmd/portico
```

Run:

```sh
go run ./cmd/portico --help
```

Test:

```sh
go test ./...
```

Format:

```sh
go fmt ./...
```

## License

GPL-3.0-or-later.
