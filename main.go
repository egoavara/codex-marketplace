package main

import (
	"embed"

	"github.com/egoavara/codex-market/cmd"
	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/jeandeaual/go-locale"
)

//go:embed locales/*.json
var localeFS embed.FS

func main() {
	// i18n 초기화
	lang := getLocale()
	i18n.Init(localeFS, lang)

	// Register plugin aliases (install, uninstall, search, update)
	cmd.RegisterPluginAliases()

	cmd.Execute()
}

// getLocale returns the locale based on config
func getLocale() string {
	configLocale := config.GetLocale()

	// If "auto", detect system locale
	if configLocale == "auto" {
		userLocale, err := locale.GetLocale()
		if err != nil || userLocale == "" {
			return "en-US"
		}
		return userLocale
	}

	// Use configured locale
	return configLocale
}
