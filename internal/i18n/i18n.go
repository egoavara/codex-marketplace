package i18n

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
)

// Init initializes the i18n bundle with the given locale files
func Init(localeFS embed.FS, lang string) error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load locale files - ignore errors for missing files
	bundle.LoadMessageFileFS(localeFS, "locales/en-us.json")
	bundle.LoadMessageFileFS(localeFS, "locales/ko-kr.json")

	localizer = i18n.NewLocalizer(bundle, lang)
	return nil
}

// T translates a message by its ID with optional template data and plural count
func T(messageID string, templateData map[string]interface{}, pluralCount ...int) string {
	config := &i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	}
	if len(pluralCount) > 0 {
		config.PluralCount = pluralCount[0]
	}

	msg, err := localizer.Localize(config)
	if err != nil {
		// Return message ID if translation fails
		return messageID
	}
	return msg
}

// SetLocale changes the current locale
func SetLocale(lang string) {
	localizer = i18n.NewLocalizer(bundle, lang)
}
