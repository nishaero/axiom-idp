package ai

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
)

// PromptTemplate represents a prompt template
type PromptTemplate struct {
	Name       string
	Template   string
	Parameters []string
}

// PromptEngine interface for building prompts
type PromptEngine interface {
	BuildQueryPrompt(query *QueryContext) string
	BuildRecommendationPrompt(query string, context map[string]string) string
	BuildIntentPrompt(query string) string
	BuildRAGPrompt(query string, context []string) string
}

// DefaultPromptEngine is the default implementation of PromptEngine
type DefaultPromptEngine struct {
	logger   *logrus.Logger
	templates map[string]*template.Template
}

// PromptTemplates contains all prompt templates
var PromptTemplates = map[string]string{
	"query": `You are an AI assistant for Axiom IDP, an Internal Developer Platform. Your role is to help developers find and understand services, tools, and resources.

User Query: {{.Query}}
User ID: {{.UserID}}
Requested Tools: {{.Tools}}
Filter: {{.Filter}}

Guidelines:
1. Be concise and focused on the developer's needs
2. Consider the user's context and recent queries
3. If tools are specified, ensure your response addresses them
4. If filters are specified, respect those constraints
5. If you don't know something, say so clearly
6. Provide actionable information when possible

Respond to the user's query in a helpful, professional manner.`,

	"recommendation": `You are recommending services in an Internal Developer Platform. Based on the following service information, explain why this service might be relevant:

Service Information:
{{.ServiceInfo}}

Additional Context:
{{.Context}}

Provide a brief, natural language explanation of why this service would be useful to the user. Keep it under 2 sentences. Focus on practical benefits and use cases.`,

	"intent": `Classify the intent of this user query into one of these categories: search, recommend, question, action, clarification.

Query: {{.Query}}

Respond with only the category name.`,

	"rag": `You have the following context from the Axiom IDP knowledge base:

{{.Context}}

User Question: {{.Query}}

Use the context to provide an accurate answer. If the context doesn't contain relevant information, acknowledge this and provide general guidance. Cite sources when applicable.`,

	"service_search": `Search for services matching these criteria:

Keywords: {{.Keywords}}
Category: {{.Category}}
Tags: {{.Tags}}
Additional Info: {{.Metadata}}

List matching services with brief descriptions.`,

	"service_comparison": `Compare these services for the following purpose: {{.Purpose}}

Service 1: {{.Service1}}
Service 2: {{.Service2}}

Provide a comparison highlighting strengths, weaknesses, and recommendations.`,
}

// NewDefaultPromptEngine creates a new default prompt engine
func NewDefaultPromptEngine() *DefaultPromptEngine {
	engine := &DefaultPromptEngine{
		logger:    logrus.New(),
		templates: make(map[string]*template.Template),
	}

	// Compile templates
	for name, tmpl := range PromptTemplates {
		t, err := template.New(name).Parse(tmpl)
		if err != nil {
			engine.logger.WithField("template", name).WithError(err).Error("Failed to parse prompt template")
			continue
		}
		engine.templates[name] = t
	}

	return engine
}

// BuildQueryPrompt builds a prompt for processing a user query
func (e *DefaultPromptEngine) BuildQueryPrompt(query *QueryContext) string {
	tmpl, exists := e.templates["query"]
	if !exists {
		return e.fallbackQueryPrompt(query)
	}

	var buf strings.Builder
	data := map[string]interface{}{
		"Query":    query.Query,
		"UserID":   query.UserID,
		"Tools":    query.Tools,
		"Filter":   query.Filter,
		"Timestamp": time.Now().Format(time.RFC3339),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		e.logger.WithError(err).Error("Failed to execute query prompt template")
		return e.fallbackQueryPrompt(query)
	}

	return buf.String()
}

// BuildRecommendationPrompt builds a prompt for generating service recommendations
func (e *DefaultPromptEngine) BuildRecommendationPrompt(query string, context map[string]string) string {
	tmpl, exists := e.templates["recommendation"]
	if !exists {
		return fmt.Sprintf("Explain why the service '%s' would be useful. Context: %v", query, context)
	}

	var buf strings.Builder
	data := map[string]interface{}{
		"ServiceInfo": query,
		"Context":     formatContext(context),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		e.logger.WithError(err).Error("Failed to execute recommendation prompt template")
		return fmt.Sprintf("Service: %s. Context: %v", query, context)
	}

	return buf.String()
}

// BuildIntentPrompt builds a prompt for intent classification
func (e *DefaultPromptEngine) BuildIntentPrompt(query string) string {
	tmpl, exists := e.templates["intent"]
	if !exists {
		return fmt.Sprintf("Classify intent: %s", query)
	}

	var buf strings.Builder
	data := map[string]interface{}{
		"Query": query,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		e.logger.WithError(err).Error("Failed to execute intent prompt template")
		return "search"
	}

	return buf.String()
}

// BuildRAGPrompt builds a prompt for retrieval-augmented generation
func (e *DefaultPromptEngine) BuildRAGPrompt(query string, context []string) string {
	tmpl, exists := e.templates["rag"]
	if !exists {
		return fmt.Sprintf("Context: %v. Question: %s", context, query)
	}

	var buf strings.Builder
	data := map[string]interface{}{
		"Query":   query,
		"Context": strings.Join(context, "\n\n"),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		e.logger.WithError(err).Error("Failed to execute RAG prompt template")
		return fmt.Sprintf("Question: %s. Context: %v", query, context)
	}

	return buf.String()
}

// fallbackQueryPrompt is a fallback for when template fails
func (e *DefaultPromptEngine) fallbackQueryPrompt(query *QueryContext) string {
	return fmt.Sprintf(
		"User query: %s. User ID: %s. Tools: %v. Filter: %v. Respond helpfully.",
		query.Query,
		query.UserID,
		query.Tools,
		query.Filter,
	)
}

// formatContext formats context map for prompt inclusion
func formatContext(context map[string]string) string {
	var parts []string
	for k, v := range context {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(parts, "\n")
}

// PromptOptimizer optimizes prompts for better LLM responses
type PromptOptimizer struct {
	logger *logrus.Logger
}

// NewPromptOptimizer creates a new prompt optimizer
func NewPromptOptimizer(logger *logrus.Logger) *PromptOptimizer {
	return &PromptOptimizer{
		logger: logger.WithField("component", "prompt_optimizer"),
	}
}

// Optimize improves a prompt for clarity and conciseness
func (o *PromptOptimizer) Optimize(prompt string, context string) string {
	// In production, would use LLM to optimize
	// For now, basic cleanup
	result := strings.TrimSpace(prompt)
	result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	result = strings.ReplaceAll(result, "  ", " ")

	return result
}

// AddExamples enhances a prompt with few-shot examples
func (o *PromptOptimizer) AddExamples(prompt string, examples []string) string {
	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n\nExamples:\n")

	for i, example := range examples {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, example))
	}

	return sb.String()
}

// ApplyChainOfThought adds chain-of-thought prompting
func (o *PromptOptimizer) ApplyChainOfThought(prompt string) string {
	return fmt.Sprintf("%s\n\nThink step by step and explain your reasoning.", prompt)
}

// ApplyTemperatureAdjustment adjusts temperature for different task types
type TaskType string

const (
	TaskCreativity   TaskType = "creativity"
	TaskAccuracy     TaskType = "accuracy"
	TaskReasoning    TaskType = "reasoning"
	TaskClassification TaskType = "classification"
)

// GetTemperature returns recommended temperature for task type
func GetTemperature(taskType TaskType) float64 {
	switch taskType {
	case TaskCreativity:
		return 0.8
	case TaskAccuracy:
		return 0.1
	case TaskReasoning:
		return 0.3
	case TaskClassification:
		return 0.0
	default:
		return 0.5
	}
}

// CreateStructuredPrompt creates a structured prompt with sections
type StructuredPrompt struct {
	Role         string
	Task         string
	Context      string
	Input        string
	OutputFormat string
	Constraints  []string
}

// Build builds the structured prompt
func (p *StructuredPrompt) Build() string {
	var sb strings.Builder

	if p.Role != "" {
		sb.WriteString(fmt.Sprintf("### ROLE\n%s\n\n", p.Role))
	}

	if p.Task != "" {
		sb.WriteString(fmt.Sprintf("### TASK\n%s\n\n", p.Task))
	}

	if p.Context != "" {
		sb.WriteString(fmt.Sprintf("### CONTEXT\n%s\n\n", p.Context))
	}

	if p.Input != "" {
		sb.WriteString(fmt.Sprintf("### INPUT\n%s\n\n", p.Input))
	}

	if len(p.Constraints) > 0 {
		sb.WriteString("### CONSTRAINTS\n")
		for _, c := range p.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	if p.OutputFormat != "" {
		sb.WriteString(fmt.Sprintf("### OUTPUT FORMAT\n%s\n", p.OutputFormat))
	}

	return sb.String()
}

// NewStructuredPrompt creates a new structured prompt
func NewStructuredPrompt() *StructuredPrompt {
	return &StructuredPrompt{}
}

// SystemPromptBuilder builds system prompts for different use cases
type SystemPromptBuilder struct {
	prompts map[string]string
}

// NewSystemPromptBuilder creates a new system prompt builder
func NewSystemPromptBuilder() *SystemPromptPromptBuilder {
	return &SystemPromptBuilder{
		prompts: make(map[string]string),
	}
}

// AddPrompt adds a custom system prompt
func (b *SystemPromptBuilder) AddPrompt(name, prompt string) {
	b.prompts[name] = prompt
}

// GetPrompt retrieves a system prompt by name
func (b *SystemPromptBuilder) GetPrompt(name string) string {
	if prompt, exists := b.prompts[name]; exists {
		return prompt
	}

	// Return default developer assistant prompt
	return `You are an AI assistant for Axiom IDP, an Internal Developer Platform that helps developers discover, deploy, and manage services. You should:

1. Be knowledgeable about software development, DevOps, and cloud technologies
2. Provide clear, actionable guidance
3. Reference specific tools, services, and best practices
4. Help developers solve problems efficiently
5. Maintain a professional, helpful tone

Available tools and resources:
- Service catalog and discovery
- CI/CD pipelines and deployments
- Kubernetes and container orchestration
- Monitoring and observability
- Security and compliance

Ask clarifying questions when needed and provide complete, helpful responses.
`
}

// CreateDeveloperPrompt creates a prompt for developer assistance
func (b *SystemPromptBuilder) CreateDeveloperPrompt() string {
	return b.GetPrompt("developer")
}

// CreateDevOpsPrompt creates a prompt for DevOps tasks
func (b *SystemPromptBuilder) CreateDevOpsPrompt() string {
	return b.GetPrompt("devops")
}

// NewSystemPromptBuilder initializes the builder
func NewSystemPromptBuilder() *SystemPromptBuilder {
	builder := &SystemPromptBuilder{
		prompts: make(map[string]string),
	}

	// Add default prompts
	builder.prompts["developer"] = `You are an AI assistant for Axiom IDP, an Internal Developer Platform. Help developers discover services, understand deployment pipelines, and resolve issues. Be technical but accessible.
`
	builder.prompts["devops"] = `You are a DevOps AI assistant for Axiom IDP. Help with CI/CD pipelines, Kubernetes deployments, infrastructure, and automation. Provide specific, actionable guidance.
`
	builder.prompts["security"] = `You are a security-focused AI assistant for Axiom IDP. Help with security best practices, compliance, vulnerability management, and secure deployment practices.
`

	return builder
}
