package models

// TemplateData holds data sent from handlers to templates
type TemplateData struct {
	Data      map[string]interface{}
	CSRFToken string
	Flash     string
	Warning   string
	Error     string
}
