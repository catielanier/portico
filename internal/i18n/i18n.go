package i18n

import (
	"embed"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

type Translator struct {
	localizer *goi18n.Localizer
}

func New(locale string) (*Translator, error) {
	bundle := goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	if _, err := bundle.LoadMessageFileFS(localeFS, "locales/en.toml"); err != nil {
		return nil, err
	}

	if locale == "" || locale == "auto" {
		locale = "en"
	}

	return &Translator{
		localizer: goi18n.NewLocalizer(bundle, locale, "en"),
	}, nil
}

func (t *Translator) T(id string, data map[string]any) string {
	msg, err := t.localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: data,
	})
	if err != nil {
		return id
	}

	return msg
}
