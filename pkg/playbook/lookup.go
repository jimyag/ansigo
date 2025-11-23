package playbook

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LookupHandler handles lookup() function calls in templates
type LookupHandler struct {
	playbookPath   string                   // Base path for resolving relative paths
	rolePathHint   string                   // Hint for role-relative paths (e.g., "roles/mihomo")
	templateEngine TemplateEngineInterface  // Template engine for rendering loaded templates
}

// NewLookupHandler creates a new lookup handler
func NewLookupHandler(playbookPath string, templateEngine TemplateEngineInterface) *LookupHandler {
	return &LookupHandler{
		playbookPath:   playbookPath,
		templateEngine: templateEngine,
	}
}

// SetRolePathHint sets a hint for resolving role-relative template paths
func (lh *LookupHandler) SetRolePathHint(rolePath string) {
	lh.rolePathHint = rolePath
}

// ProcessLookups processes all lookup() calls in a template string
// This should be called BEFORE Jinja2 rendering
func (lh *LookupHandler) ProcessLookups(template string, context map[string]interface{}) (string, error) {
	// Pattern: {{ lookup('template', 'path/to/file') }}
	// Also supports: lookup("template", "path/to/file") without {{ }}
	// We need to handle both cases

	// Try to match {{ lookup(...) }} first (most common case)
	wrappedPattern := regexp.MustCompile(`\{\{\s*lookup\(['"]template['"],\s*['"]([^'"]+)['"]\)\s*\}\}`)
	result := template

	// Process wrapped lookups first
	wrappedMatches := wrappedPattern.FindAllStringSubmatch(template, -1)
	for _, match := range wrappedMatches {
		fullMatch := match[0]   // The entire {{ lookup(...) }} expression
		templatePath := match[1] // The template file path

		// Load and process the template file
		content, err := lh.loadTemplate(templatePath, context)
		if err != nil {
			return "", fmt.Errorf("lookup('template', '%s') failed: %w", templatePath, err)
		}

		// Replace the entire {{ lookup(...) }} with the rendered content
		result = strings.Replace(result, fullMatch, content, 1)
	}

	// Also handle bare lookup() calls (without {{ }})
	barePattern := regexp.MustCompile(`lookup\(['"]template['"],\s*['"]([^'"]+)['"]\)`)
	bareMatches := barePattern.FindAllStringSubmatch(result, -1)

	for _, match := range bareMatches {
		fullMatch := match[0]   // The entire lookup(...) expression
		templatePath := match[1] // The template file path

		// Load and process the template file
		content, err := lh.loadTemplate(templatePath, context)
		if err != nil {
			return "", fmt.Errorf("lookup('template', '%s') failed: %w", templatePath, err)
		}

		// Replace the lookup() call with the template content
		result = strings.Replace(result, fullMatch, content, 1)
	}

	return result, nil
}

// loadTemplate loads a template file and optionally renders it
func (lh *LookupHandler) loadTemplate(templatePath string, context map[string]interface{}) (string, error) {
	// Resolve the full path to the template file
	// Try multiple search paths:
	// 1. Role templates directory (if role hint is set)
	// 2. Playbook templates directory
	// 3. Absolute path

	// Get the playbook directory (playbookPath might be a file path)
	playbookDir := lh.playbookPath
	if filepath.Ext(playbookDir) != "" {
		// It's a file, get the directory
		playbookDir = filepath.Dir(playbookDir)
	}

	var searchPaths []string

	// Add role templates path if available
	if lh.rolePathHint != "" {
		roleTemplatesPath := filepath.Join(playbookDir, lh.rolePathHint, "templates", templatePath)
		searchPaths = append(searchPaths, roleTemplatesPath)
	}

	// Add playbook templates path
	playbookTemplatesPath := filepath.Join(playbookDir, "templates", templatePath)
	searchPaths = append(searchPaths, playbookTemplatesPath)

	// Add playbook directory itself
	playbookRelativePath := filepath.Join(playbookDir, templatePath)
	searchPaths = append(searchPaths, playbookRelativePath)

	// Try absolute path
	if filepath.IsAbs(templatePath) {
		searchPaths = append(searchPaths, templatePath)
	}

	// Try each path until we find the file
	var content []byte
	var err error
	foundPath := ""

	for _, path := range searchPaths {
		content, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if foundPath == "" {
		return "", fmt.Errorf("template file '%s' not found in any of: %v", templatePath, searchPaths)
	}

	// Render the template content with the current context
	if lh.templateEngine != nil {
		rendered, err := lh.templateEngine.RenderString(string(content), context)
		if err != nil {
			return "", fmt.Errorf("failed to render template '%s': %w", templatePath, err)
		}
		return rendered, nil
	}

	// Fallback: return raw content if no template engine
	return string(content), nil
}

// ProcessLookupsInVars processes lookup() calls in all variables
func (lh *LookupHandler) ProcessLookupsInVars(vars map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range vars {
		switch v := value.(type) {
		case string:
			// Process lookup() calls in string values
			processed, err := lh.ProcessLookups(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to process lookups in var '%s': %w", key, err)
			}
			result[key] = processed

		case map[string]interface{}:
			// Recursively process nested maps
			processed, err := lh.ProcessLookupsInVars(v, context)
			if err != nil {
				return nil, err
			}
			result[key] = processed

		case []interface{}:
			// Process arrays
			processedArray := make([]interface{}, len(v))
			for i, item := range v {
				if strItem, ok := item.(string); ok {
					processed, err := lh.ProcessLookups(strItem, context)
					if err != nil {
						return nil, fmt.Errorf("failed to process lookups in array item: %w", err)
					}
					processedArray[i] = processed
				} else {
					processedArray[i] = item
				}
			}
			result[key] = processedArray

		default:
			result[key] = value
		}
	}

	return result, nil
}
