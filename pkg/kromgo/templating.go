package kromgo

import (
	"fmt"
	"text/template"
)

var templates = template.FuncMap{
	"simplifyDate": simplifyDate,
}

func simplifyDate(days int) string {
	years := days / 365
	days %= 365
	months := days / 30
	days %= 30

	result := ""
	if years > 0 {
		result += fmt.Sprintf("%dy", years)
	}
	if months > 0 {
		result += fmt.Sprintf("%dm", months)
	}
	if days > 0 {
		result += fmt.Sprintf("%dd", days)
	}
	if result == "" {
		result = "0d"
	}
	return result
}
