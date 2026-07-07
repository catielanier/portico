package jokes

import (
	"math/rand"
	"strings"
	"time"
)

type Context struct {
	Command  string
	Atom     string
	Category string
	Package  string
	Features []string
}

type Joke struct {
	Key    string
	Scopes []string
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

var defaultJokes = []Joke{
	{Key: "joke.arch", Scopes: []string{"any"}},
	{Key: "joke.lmgtfy", Scopes: []string{"any"}},
	{Key: "joke.old_forum", Scopes: []string{"any"}},
	{Key: "joke.judge_make_conf", Scopes: []string{"any"}},
	{Key: "joke.ceiling_cat", Scopes: []string{"any"}},
	{Key: "joke.init_flamewar", Scopes: []string{"any"}},
	{Key: "joke.surprise_desktop", Scopes: []string{"any"}},
	{Key: "joke.alias_vim_to_nano", Scopes: []string{"any"}},
	{Key: "joke.just_use_ubuntu", Scopes: []string{"any"}},
	{Key: "joke.mirror_stress_toy", Scopes: []string{"any"}},

	{Key: "joke.just_use_rust", Scopes: []string{"atom:dev-lang/go"}},
	{Key: "joke.linkedin_rust", Scopes: []string{"atom:dev-lang/rust"}},
	{Key: "joke.wordpress_necromancy", Scopes: []string{"atom:dev-lang/php"}},
	{Key: "joke.java_time_machine", Scopes: []string{"atom:dev-lang/java"}},
	{Key: "joke.obs_poggers", Scopes: []string{"atom:media-video/obs-studio"}},

	{Key: "joke.st_anger_snare", Scopes: []string{"category:media-sound", "category:media-video"}},
	{Key: "joke.bios_therapist", Scopes: []string{"category:sys-kernel", "feature:bootloader"}},
	{Key: "joke.overlay_goblins", Scopes: []string{"feature:overlay", "feature:repo"}},
}

func RandomKey(ctx Context) string {
	ctx = normalizeContext(ctx)

	pool := mostSpecificPool(ctx)
	if len(pool) == 0 {
		return "joke.ceiling_cat"
	}

	return pool[rng.Intn(len(pool))].Key
}

func normalizeContext(ctx Context) Context {
	if ctx.Atom != "" && (ctx.Category == "" || ctx.Package == "") {
		category, pkg := splitAtom(ctx.Atom)

		if ctx.Category == "" {
			ctx.Category = category
		}

		if ctx.Package == "" {
			ctx.Package = pkg
		}
	}

	return ctx
}

func mostSpecificPool(ctx Context) []Joke {
	if ctx.Atom != "" {
		if pool := filterByScope("atom:" + ctx.Atom); len(pool) > 0 {
			return pool
		}
	}

	if ctx.Category != "" {
		if pool := filterByScope("category:" + ctx.Category); len(pool) > 0 {
			return pool
		}
	}

	for _, feature := range ctx.Features {
		if pool := filterByScope("feature:" + feature); len(pool) > 0 {
			return pool
		}
	}

	if ctx.Command != "" {
		if pool := filterByScope("command:" + ctx.Command); len(pool) > 0 {
			return pool
		}
	}

	return filterByScope("any")
}

func filterByScope(scope string) []Joke {
	var out []Joke

	for _, joke := range defaultJokes {
		for _, candidate := range joke.Scopes {
			if candidate == scope {
				out = append(out, joke)
				break
			}
		}
	}

	return out
}

func splitAtom(atom string) (string, string) {
	parts := strings.SplitN(atom, "/", 2)
	if len(parts) != 2 {
		return "", atom
	}

	return parts[0], parts[1]
}
