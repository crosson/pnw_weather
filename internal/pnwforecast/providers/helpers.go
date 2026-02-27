package providers

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func normalizeHTMLText(raw string) string {
	withoutScripts := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(raw, " ")
	withoutStyles := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(withoutScripts, " ")
	withoutTags := regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(withoutStyles, " ")
	unescaped := html.UnescapeString(withoutTags)
	spaceCollapsed := regexp.MustCompile(`\s+`).ReplaceAllString(unescaped, " ")
	return strings.TrimSpace(spaceCollapsed)
}

func toTitleWords(v string) string {
	parts := strings.Fields(v)
	for i := range parts {
		runes := []rune(parts[i])
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
