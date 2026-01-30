package parser

import (
	"fmt"
	"strings"
)

// Registry manages parser registration and format auto-detection.
// It maintains an ordered list of parsers and can automatically
// detect the appropriate parser for a log line.
type Registry struct {
	// parsers holds all registered parsers in priority order.
	parsers []Parser

	// cached stores the auto-detected parser after first successful match.
	// Used in strict mode to avoid re-detection on every line.
	cached Parser

	// adaptive determines detection behavior:
	// - true: re-detect format for each line (mixed formats)
	// - false: cache first detected format (strict mode, default)
	adaptive bool

	// forcedFormat specifies a parser by name, skipping auto-detection.
	forcedFormat string
}

// RegistryOption configures the Registry.
type RegistryOption func(*Registry)

// WithAdaptiveMode enables re-detection for each line.
// Use this when processing logs with mixed formats.
func WithAdaptiveMode() RegistryOption {
	return func(r *Registry) {
		r.adaptive = true
	}
}

// WithForcedFormat specifies a parser by name, skipping auto-detection.
func WithForcedFormat(format string) RegistryOption {
	return func(r *Registry) {
		r.forcedFormat = strings.ToLower(format)
	}
}

// NewRegistry creates a new parser registry with default parsers.
// Parsers are registered in priority order (first match wins).
func NewRegistry(opts ...RegistryOption) *Registry {
	r := &Registry{
		parsers: make([]Parser, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	// Register built-in parsers in priority order.
	// JSON first (already structured), then more specific formats.
	r.Register(NewJSONParser())
	r.Register(NewKeyValueParser())
	r.Register(NewSyslogParser())
	r.Register(NewApacheParser())
	r.Register(NewGenericParser())

	return r
}

// Register adds a parser to the registry.
// Parsers are tried in the order they are registered.
func (r *Registry) Register(p Parser) {
	r.parsers = append(r.parsers, p)
}

// GetParser returns the parser for the given format name.
// Returns nil if no parser with that name is registered.
func (r *Registry) GetParser(name string) Parser {
	name = strings.ToLower(name)
	for _, p := range r.parsers {
		if strings.ToLower(p.Name()) == name {
			return p
		}
	}
	return nil
}

// ListParsers returns information about all registered parsers.
func (r *Registry) ListParsers() []struct {
	Name        string
	Description string
} {
	result := make([]struct {
		Name        string
		Description string
	}, len(r.parsers))

	for i, p := range r.parsers {
		result[i].Name = p.Name()
		result[i].Description = p.Description()
	}
	return result
}

// Parse parses a log line using the appropriate parser.
// Uses forced format if specified, otherwise auto-detects.
func (r *Registry) Parse(line string) (*Entry, error) {
	// Handle empty lines
	if strings.TrimSpace(line) == "" {
		entry := NewEntry(line)
		entry.ParseError = ErrEmptyLine
		return entry, nil
	}

	// Use forced format if specified
	if r.forcedFormat != "" {
		parser := r.GetParser(r.forcedFormat)
		if parser == nil {
			return nil, fmt.Errorf("unknown format: %s", r.forcedFormat)
		}
		return parser.Parse(line)
	}

	// Use cached parser in strict mode
	if !r.adaptive && r.cached != nil {
		return r.cached.Parse(line)
	}

	// Auto-detect: try each parser until one succeeds
	for _, p := range r.parsers {
		if p.CanParse(line) {
			entry, err := p.Parse(line)
			if err == nil && entry.ParseError == nil {
				// Cache successful parser in strict mode
				if !r.adaptive && r.cached == nil {
					r.cached = p
				}
				return entry, nil
			}
		}
	}

	// Fallback: use generic parser (always succeeds)
	generic := r.GetParser("generic")
	if generic != nil {
		return generic.Parse(line)
	}

	// Last resort: wrap as raw
	entry := NewEntry(line)
	entry.Fields["raw"] = line
	entry.ParseError = ErrNoMatch
	return entry, nil
}
