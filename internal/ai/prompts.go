package ai

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
)

// PromptTemplate represents a named prompt template.
type PromptTemplate struct {
	Name       string
	Template   string
	Parameters []string
}

// DefaultPromptEngine renders the prompts used by the AI engine.
type DefaultPromptEngine struct {
	logger    *logrus.Logger
	templates map[string]*template.Template
}

// PromptTemplates contains the built-in prompt templates.
var PromptTemplates = map[string]string{
	"query": `You are an AI assistant for Axiom IDP.

User Query: {{.Query}}
User ID: {{.UserID}}
Requested Tools: {{.Tools}}
Filter: {{.Filter}}

Respond clearly, concisely, and with action-oriented guidance.`,
	"recommendation": `You are recommending services in an Internal Developer Platform.

Service Information:
{{.ServiceInfo}}

Additional Context:
{{.Context}}

Explain why this service is relevant in one or two sentences.`,
	"intent": `Classify the intent of this user query into one of: search, recommend, question, action, clarification.

Query: {{.Query}}

Respond with only the category name.`,
}

// NewDefaultPromptEngine creates a prompt engine with compiled templates.
func NewDefaultPromptEngine() *DefaultPromptEngine {
	engine := &DefaultPromptEngine{
		logger:    logrus.New(),
		templates: make(map[string]*template.Template),
	}

	for name, tmpl := range PromptTemplates {
		parsed, err := template.New(name).Parse(tmpl)
		if err != nil {
			engine.logger.WithField("template", name).WithError(err).Error("failed to parse prompt template")
			continue
		}
		engine.templates[name] = parsed
	}

	return engine
}

// BuildQueryPrompt builds a prompt for processing a query.
func (e *DefaultPromptEngine) BuildQueryPrompt(query *QueryContext) string {
	tmpl, ok := e.templates["query"]
	if !ok {
		return fallbackQueryPrompt(query)
	}

	var sb strings.Builder
	data := map[string]interface{}{
		"Query":  query.Query,
		"UserID": query.UserID,
		"Tools":  query.Tools,
		"Filter": query.Filter,
	}

	if err := tmpl.Execute(&sb, data); err != nil {
		e.logger.WithError(err).Error("failed to execute query prompt template")
		return fallbackQueryPrompt(query)
	}
	return sb.String()
}

// BuildRecommendationPrompt builds a prompt for recommendation reasons.
func (e *DefaultPromptEngine) BuildRecommendationPrompt(query string, context map[string]string) string {
	tmpl, ok := e.templates["recommendation"]
	if !ok {
		return fmt.Sprintf("Explain why the service %q would be useful. Context: %v", query, context)
	}

	var sb strings.Builder
	data := map[string]interface{}{
		"ServiceInfo": query,
		"Context":     formatContext(context),
	}

	if err := tmpl.Execute(&sb, data); err != nil {
		e.logger.WithError(err).Error("failed to execute recommendation prompt template")
		return fmt.Sprintf("Service: %s. Context: %v", query, context)
	}
	return sb.String()
}

// BuildIntentPrompt builds a prompt for intent classification.
func (e *DefaultPromptEngine) BuildIntentPrompt(query string) string {
	tmpl, ok := e.templates["intent"]
	if !ok {
		return fmt.Sprintf("Classify intent: %s", query)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, map[string]interface{}{"Query": query}); err != nil {
		e.logger.WithError(err).Error("failed to execute intent prompt template")
		return "search"
	}
	return sb.String()
}

func fallbackQueryPrompt(query *QueryContext) string {
	return fmt.Sprintf("User query: %s. User ID: %s. Tools: %v. Filter: %v.", query.Query, query.UserID, query.Tools, query.Filter)
}

func formatContext(context map[string]string) string {
	if len(context) == 0 {
		return ""
	}

	parts := make([]string, 0, len(context))
	for k, v := range context {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(parts, "\n")
}
