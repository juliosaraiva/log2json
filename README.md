# log2json

A streaming CLI tool that converts log files to JSON in real-time. Perfect for piping logs through Unix pipelines.

## Features

- **Real-time streaming**: Process logs as they arrive (works with `tail -f`)
- **Auto-detection**: Automatically detects log format (syslog, apache, json, key-value)
- **Custom patterns**: Define your own regex patterns with named groups
- **NDJSON output**: One JSON object per line, perfect for piping to `jq`
- **Zero dependencies**: Single binary, no runtime required

## Installation

### From source

```bash
# Clone or download the source
git clone https://github.com/youruser/log2json.git
cd log2json

# Build
go build -o log2json ./cmd/log2json

# Or use make
make build
```

### Pre-built binaries

Download from the releases page (when available).

## Usage

```bash
# Basic usage - auto-detect format
tail -f /var/log/syslog | log2json

# Force specific format
cat access.log | log2json --format=apache

# Custom regex pattern
cat app.log | log2json --pattern='(?P<ts>\S+) \[(?P<level>\w+)\] (?P<msg>.*)'

# Add metadata fields
cat app.log | log2json --add-timestamp --add-line-number

# Select specific fields
cat access.log | log2json -f apache -F ip,status,path

# Chain with jq for filtering
tail -f app.log | log2json | jq 'select(.level == "ERROR")'
```

## Supported Formats

| Format | Description | Example |
|--------|-------------|---------|
| `json` | Already JSON formatted | `{"level":"info","msg":"hello"}` |
| `kv` | Key=value pairs (logfmt) | `level=info msg="hello world"` |
| `syslog` | Standard syslog | `Jan 15 10:30:45 host prog[123]: message` |
| `apache` | Apache/Nginx combined | `192.168.1.1 - - [15/Jan/2024:10:30:45 +0000] "GET /" 200` |
| `generic` | Timestamp + level patterns | `2024-01-15 INFO Hello world` |

## Options

```
Parser Options:
  -f, --format <FORMAT>     Force specific format (auto-detect if empty)
  -p, --pattern <REGEX>     Custom regex with named groups
  --adaptive                Re-detect format for each line

Output Options:
  --pretty                  Pretty-print JSON (not for pipes)
  -F, --fields <FIELDS>     Only output these fields (comma-separated)
  --add-timestamp           Add _ingestTime field
  --add-line-number         Add _lineNumber field
  --add-raw                 Add _raw field with original line
  --omit-empty              Skip entries with parse errors

General:
  -q, --quiet               Suppress warnings
  -v, --verbose             Debug output
  -l, --list                List available formats
  -h, --help                Show help
  -V, --version             Show version
```

## Examples

### Syslog to JSON

**Input:**
```
Jan 15 10:30:45 myhost sshd[1234]: Accepted password for user from 192.168.1.1
```

**Output:**
```json
{"timestamp":"Jan 15 10:30:45","host":"myhost","program":"sshd","pid":1234,"message":"Accepted password for user from 192.168.1.1"}
```

### Apache Logs

**Input:**
```
192.168.1.1 - john [15/Jan/2024:10:30:45 +0000] "GET /index.html HTTP/1.1" 200 1234 "http://ref.com" "Mozilla/5.0"
```

**Output:**
```json
{"ip":"192.168.1.1","user":"john","timestamp":"15/Jan/2024:10:30:45 +0000","method":"GET","path":"/index.html","protocol":"HTTP/1.1","status":200,"size":1234,"referer":"http://ref.com","useragent":"Mozilla/5.0"}
```

### Key-Value Logs

**Input:**
```
time=2024-01-15T10:30:45Z level=info msg="Server started" port=8080
```

**Output:**
```json
{"time":"2024-01-15T10:30:45Z","level":"info","msg":"Server started","port":8080}
```

### Custom Pattern

```bash
# Log format: "[2024-01-15 10:30:45] ERROR in module: Something failed"
cat app.log | log2json -p '\[(?P<timestamp>[^\]]+)\] (?P<level>\w+) in (?P<module>\w+): (?P<message>.*)'
```

**Output:**
```json
{"timestamp":"2024-01-15 10:30:45","level":"ERROR","module":"module","message":"Something failed"}
```

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌────────────┐
│   STDIN     │───▶│   STREAM     │───▶│   FORMAT    │───▶│   JSON     │
│   INPUT     │    │   READER     │    │   PARSER    │    │   EMITTER  │
└─────────────┘    └──────────────┘    └─────────────┘    └────────────┘
                                              │
                                              ▼
                                       ┌─────────────┐
                                       │   FORMAT    │
                                       │   REGISTRY  │
                                       └─────────────┘
```

## Project Structure

```
log2json/
├── cmd/
│   └── log2json/
│       └── main.go           # CLI entry point
├── internal/
│   ├── parser/
│   │   ├── parser.go         # Parser interface
│   │   ├── registry.go       # Format auto-detection
│   │   ├── json_parser.go    # JSON format
│   │   ├── keyvalue_parser.go # Key=value format
│   │   ├── syslog_parser.go  # Syslog format
│   │   ├── apache_parser.go  # Apache format
│   │   ├── generic_parser.go # Generic fallback
│   │   └── regex_parser.go   # Custom regex
│   ├── reader/
│   │   └── reader.go         # Stdin line reader
│   └── emitter/
│       └── emitter.go        # JSON output
├── testdata/                 # Sample log files
├── go.mod
├── Makefile
└── README.md
```

## License

MIT License
