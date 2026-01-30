// log2json converts streaming log input to JSON output.
// Reads from stdin, writes to stdout. Designed for Unix pipes.
//
// Usage:
//
//	tail -f /var/log/syslog | log2json
//	cat access.log | log2json --format=apache
//	cat app.log | log2json --pattern='(?P<ts>\S+) (?P<level>\w+) (?P<msg>.*)'
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/juliosaraiva/log2json/internal/emitter"
	"github.com/juliosaraiva/log2json/internal/parser"
	"github.com/juliosaraiva/log2json/internal/reader"
)

// Version information (set via build flags)
var version = "dev"

// Config holds all CLI configuration options.
type Config struct {
	// Parser options
	Format   string // Force specific format
	Pattern  string // Custom regex pattern
	Adaptive bool   // Re-detect format per line

	// Output options
	Pretty        bool     // Pretty-print JSON
	Fields        []string // Only output these fields
	AddTimestamp  bool     // Add _ingestTime field
	AddLineNumber bool     // Add _lineNumber field
	AddRaw        bool     // Add _raw field
	OmitEmpty     bool     // Skip entries with parse errors

	// General options
	Quiet   bool // Suppress warnings
	Verbose bool // Debug output
	List    bool // List available formats
	Help    bool // Show help
	Version bool // Show version
}

func main() {
	cfg := parseFlags()

	// Handle info flags
	if cfg.Version {
		fmt.Printf("log2json version %s\n", version)
		os.Exit(0)
	}

	if cfg.Help {
		printUsage()
		os.Exit(0)
	}

	if cfg.List {
		listFormats()
		os.Exit(0)
	}

	// Run the converter
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags parses command line arguments into Config.
func parseFlags() Config {
	var cfg Config
	var fieldsStr string

	// Parser options
	flag.StringVar(&cfg.Format, "format", "", "Force log format (auto-detect if empty)")
	flag.StringVar(&cfg.Format, "f", "", "Force log format (shorthand)")
	flag.StringVar(&cfg.Pattern, "pattern", "", "Custom regex with named groups")
	flag.StringVar(&cfg.Pattern, "p", "", "Custom regex (shorthand)")
	flag.BoolVar(&cfg.Adaptive, "adaptive", false, "Re-detect format for each line")

	// Output options
	flag.BoolVar(&cfg.Pretty, "pretty", false, "Pretty-print JSON output")
	flag.StringVar(&fieldsStr, "fields", "", "Only output these fields (comma-separated)")
	flag.StringVar(&fieldsStr, "F", "", "Only output these fields (shorthand)")
	flag.BoolVar(&cfg.AddTimestamp, "add-timestamp", false, "Add _ingestTime field")
	flag.BoolVar(&cfg.AddLineNumber, "add-line-number", false, "Add _lineNumber field")
	flag.BoolVar(&cfg.AddRaw, "add-raw", false, "Add _raw field with original line")
	flag.BoolVar(&cfg.OmitEmpty, "omit-empty", false, "Skip entries with parse errors")

	// General options
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress warnings to stderr")
	flag.BoolVar(&cfg.Quiet, "q", false, "Suppress warnings (shorthand)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Debug output to stderr")
	flag.BoolVar(&cfg.Verbose, "v", false, "Debug output (shorthand)")
	flag.BoolVar(&cfg.List, "list", false, "List available formats")
	flag.BoolVar(&cfg.List, "l", false, "List formats (shorthand)")
	flag.BoolVar(&cfg.Help, "help", false, "Show help")
	flag.BoolVar(&cfg.Help, "h", false, "Show help (shorthand)")
	flag.BoolVar(&cfg.Version, "version", false, "Show version")
	flag.BoolVar(&cfg.Version, "V", false, "Show version (shorthand)")

	// Custom usage message
	flag.Usage = printUsage

	flag.Parse()

	// Parse fields list
	if fieldsStr != "" {
		cfg.Fields = strings.Split(fieldsStr, ",")
		for i := range cfg.Fields {
			cfg.Fields[i] = strings.TrimSpace(cfg.Fields[i])
		}
	}

	return cfg
}

// printUsage prints the help message.
func printUsage() {
	fmt.Fprintf(os.Stderr, `log2json - Convert log streams to JSON in real-time

USAGE:
    log2json [OPTIONS]
    <command> | log2json [OPTIONS]

OPTIONS:
    -f, --format <FORMAT>     Force specific format (auto-detect if empty)
                              Use --list to see available formats
    -p, --pattern <REGEX>     Custom regex with named groups
                              Example: '(?P<time>\S+) (?P<level>\w+) (?P<msg>.*)'
    --adaptive                Re-detect format for each line (for mixed logs)

    --pretty                  Pretty-print JSON (not recommended for pipes)
    -F, --fields <FIELDS>     Only output these fields (comma-separated)
    --add-timestamp           Add _ingestTime field with ingestion time
    --add-line-number         Add _lineNumber field
    --add-raw                 Add _raw field with original line
    --omit-empty              Skip entries with parse errors

    -q, --quiet               Suppress warnings to stderr
    -v, --verbose             Debug output to stderr
    -l, --list                List available formats
    -h, --help                Show this help
    -V, --version             Show version

EXAMPLES:
    # Auto-detect format from syslog
    tail -f /var/log/syslog | log2json

    # Parse Apache access logs
    cat access.log | log2json -f apache

    # Custom pattern for application logs
    cat app.log | log2json -p '(?P<ts>\d{4}-\d{2}-\d{2}) (?P<level>\w+): (?P<msg>.*)'

    # Filter errors with jq
    cat app.log | log2json | jq 'select(.level == "ERROR")'

    # Add metadata and select fields
    cat app.log | log2json --add-timestamp -F timestamp,level,message

`)
}

// listFormats prints available log formats.
func listFormats() {
	registry := parser.NewRegistry()
	fmt.Println("Available log formats:")
	fmt.Println()
	for _, p := range registry.ListParsers() {
		fmt.Printf("  %-12s  %s\n", p.Name, p.Description)
	}
	fmt.Println()
	fmt.Println("Use -f/--format to force a specific format, or omit for auto-detection.")
}

// run executes the main conversion pipeline using stdin/stdout/stderr.
func run(cfg Config) error {
	return runPipeline(cfg, os.Stdin, os.Stdout, os.Stderr)
}

// runPipeline executes the conversion pipeline with explicit I/O.
func runPipeline(cfg Config, input io.Reader, output io.Writer, errOutput io.Writer) error {
	// Build parser registry options
	var regOpts []parser.RegistryOption

	if cfg.Format != "" {
		regOpts = append(regOpts, parser.WithForcedFormat(cfg.Format))
	}
	if cfg.Adaptive {
		regOpts = append(regOpts, parser.WithAdaptiveMode())
	}

	// Create registry
	registry := parser.NewRegistry(regOpts...)

	// Validate format exists (fail fast instead of per-line errors)
	if cfg.Format != "" && cfg.Pattern == "" {
		if registry.GetParser(cfg.Format) == nil {
			return fmt.Errorf("unknown format %q; use --list to see available formats", cfg.Format)
		}
	}

	// Handle custom pattern
	if cfg.Pattern != "" {
		regexParser, err := parser.NewRegexParser(cfg.Pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
		// Insert custom parser at highest priority
		registry = parser.NewRegistry(parser.WithForcedFormat("regex"))
		registry.Register(regexParser)
	}

	// Create emitter
	emitOpts := emitter.Options{
		Pretty:        cfg.Pretty,
		Fields:        cfg.Fields,
		AddTimestamp:  cfg.AddTimestamp,
		AddLineNumber: cfg.AddLineNumber,
		AddRaw:        cfg.AddRaw,
		OmitEmpty:     cfg.OmitEmpty,
	}
	emit := emitter.New(output, emitOpts)
	defer emit.Close()

	// Create stream reader
	streamReader := reader.New(input)

	// Process lines
	lineCount := 0
	errorCount := 0

	for line := range streamReader.Lines() {
		lineCount++

		// Handle read errors
		if line.Err != nil {
			if !cfg.Quiet {
				fmt.Fprintf(errOutput, "read error at line %d: %v\n", line.Number, line.Err)
			}
			errorCount++
			continue
		}

		// Parse the line
		entry, err := registry.Parse(line.Text)
		if err != nil {
			if !cfg.Quiet {
				fmt.Fprintf(errOutput, "parse error at line %d: %v\n", line.Number, err)
			}
			errorCount++
			continue
		}

		// Set line number
		entry.LineNum = line.Number

		// Emit JSON
		if err := emit.Emit(entry); err != nil {
			if !cfg.Quiet {
				fmt.Fprintf(errOutput, "output error at line %d: %v\n", line.Number, err)
			}
			errorCount++
		}
	}

	// Print summary in verbose mode
	if cfg.Verbose {
		fmt.Fprintf(errOutput, "processed %d lines, %d errors\n", lineCount, errorCount)
	}

	return nil
}
