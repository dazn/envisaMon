package main

import (
	"envisaMon/tpi"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// 1. Parse command-line arguments and flags
	config, err := parseArgs(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// 2. Read password from environment
	password := os.Getenv("ENVISALINK_TPI_KEY")
	if password == "" {
		fmt.Fprintln(os.Stderr, "ERROR: ENVISALINK_TPI_KEY environment variable not set")
		os.Exit(1)
	}

	// 3. Set up dual logging with lumberjack
	tpiLogger, appLogger, err := setupLogging(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to set up logging: %v\n", err)
		os.Exit(1)
	}
	appLogger.Printf("DEBUG: Parsed URL from arg: %s (path: %s)", config.DestinationURL, config.DestinationPath)

	// 4. Create TPI client
	client := tpi.NewClient(
		fmt.Sprintf("%s:%d", config.EnvisaLinkIP, config.EnvisaLinkPort),
		password,
		tpiLogger,
		appLogger,
		config.DeduplicateLimit,
	)

	// 5. Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		appLogger.Println("INFO: Shutting down...")
		client.Close()
		os.Exit(0)
	}()

	// 6. Main monitoring loop with auto-reconnect
	appLogger.Printf("INFO: Starting TPI monitor for %s", config.EnvisaLinkIP)
	for {
		err := client.Connect()
		if err != nil {
			// Connection or auth failed, will retry with backoff
			continue
		}

		// Connection established and authenticated
		// ReadLoop() runs until error or disconnect
		err = client.ReadLoop()
		if err != nil {
			appLogger.Printf("WARN: Connection lost: %v", err)
			// Will reconnect with exponential backoff
		}
	}
}

type Config struct {
	EnvisaLinkIP     string
	EnvisaLinkPort   int
	DestinationURL   string
	DestinationPath  string
	Verbose          bool
	Deduplicate      bool
	DeduplicateLimit int
}

func parseArgs(args []string) (*Config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "Usage: %s [options] <ip>[:port] [<url>]\n", os.Args[0])
		fmt.Fprintf(out, "\nArguments:\n")
		fmt.Fprintf(out, "  <ip>[:port]    EnvisaLink IP address, optionally with port (default: 4025)\n")
		fmt.Fprintf(out, "  [<url>]        Optional: Event destination URL (must be https)\n")
		fmt.Fprintf(out, "\nOptions:\n")
		fs.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  %s -v 192.168.1.100\n", os.Args[0])
		fmt.Fprintf(out, "  %s -u 192.168.1.100\n", os.Args[0])
		fmt.Fprintf(out, "  %s -u 100 192.168.1.100\n", os.Args[0])
		fmt.Fprintf(out, "  %s 192.168.1.100 https://events.example.com:8080\n", os.Args[0])
	}
	return parseConfig(fs, args)
}

func parseConfig(fs *flag.FlagSet, args []string) (*Config, error) {
	config := &Config{}
	fs.BoolVar(&config.Verbose, "v", false, "verbose output (print logs and TPI messages to stdout)")
	fs.BoolVar(&config.Deduplicate, "u", false, "deduplicate consecutive identical TPI messages. Optionally specify number of duplicates to ignore (e.g., -u 10)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	config.DeduplicateLimit = -1 // Default: disabled
	argOffset := 0

	if config.Deduplicate {
		config.DeduplicateLimit = 0 // Default for -u: infinite
		// Check if the next argument is a number (deduplication limit)
		if fs.NArg() > 0 {
			limit, err := strconv.Atoi(fs.Arg(0))
			if err == nil && limit >= 0 {
				config.DeduplicateLimit = limit
				argOffset = 1
			}
		}
	}

	if fs.NArg()-argOffset < 1 || fs.NArg()-argOffset > 2 {
		fs.Usage()
		return nil, fmt.Errorf("expected 1 or 2 positional arguments, got %d", fs.NArg()-argOffset)
	}

	// Parse first argument: <ip>[:port]
	ipPortArg := fs.Arg(argOffset)
	config.EnvisaLinkPort = 4025 // default port

	if strings.Contains(ipPortArg, ":") {
		host, portStr, err := net.SplitHostPort(ipPortArg)
		if err != nil {
			fs.Usage()
			return nil, fmt.Errorf("invalid IP:port format '%s': %w", ipPortArg, err)
		}
		config.EnvisaLinkIP = host
		port, err := strconv.Atoi(portStr)
		if err != nil {
			fs.Usage()
			return nil, fmt.Errorf("invalid port number in '%s': %s", ipPortArg, portStr)
		}
		config.EnvisaLinkPort = port
	} else {
		config.EnvisaLinkIP = ipPortArg
	}

	if config.EnvisaLinkPort < 1 || config.EnvisaLinkPort > 65535 {
		fs.Usage()
		return nil, fmt.Errorf("port must be between 1 and 65535, got: %d", config.EnvisaLinkPort)
	}

	// Parse second argument if present: URL in format https://host:port/path
	if fs.NArg()-argOffset == 2 {
		urlArg := fs.Arg(argOffset + 1)
		config.DestinationURL = urlArg
		parsedURL, err := url.Parse(urlArg)
		if err != nil {
			fs.Usage()
			return nil, fmt.Errorf("invalid URL format: %w", err)
		}

		// Validate URL scheme
		if parsedURL.Scheme != "https" {
			fs.Usage()
			return nil, fmt.Errorf("URL scheme must be 'https', got: '%s'", parsedURL.Scheme)
		}

		// Validate URL host is present
		if parsedURL.Host == "" {
			fs.Usage()
			return nil, fmt.Errorf("URL must include a host")
		}

		// Extract the URL path (empty string if not present)
		config.DestinationPath = parsedURL.Path
	}

	return config, nil
}


func setupLogging(config *Config) (*log.Logger, *log.Logger, error) {
	// Ensure logs directory exists
	if err := os.MkdirAll("./logs", 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// TPI message logger (raw messages only, NO PREFIX/TIMESTAMP)
	tpiRoller := &lumberjack.Logger{
		Filename:   "./logs/tpi-messages.log",
		MaxSize:    5, // megabytes
		MaxBackups: 3, // keep only the 3 most recent log files
		Compress:   true,
	}

	// Application event logger
	appRoller := &lumberjack.Logger{
		Filename:   "./logs/application.log",
		MaxSize:    5, // megabytes
		MaxBackups: 3, // keep only the 3 most recent log files
		Compress:   true,
	}

	// Prepare SystemID
	systemID := fmt.Sprintf("%s:%d", config.EnvisaLinkIP, config.EnvisaLinkPort)

	// Remote Reporters
	// Only enabled if URL is provided and ALARM_MON_API_KEY is set
	var tpiReporter, appReporter *AsyncReporter
	apiKey := os.Getenv("ALARM_MON_API_KEY")
	if config.DestinationURL != "" && apiKey != "" {
		tpiReporter = NewAsyncReporter(config.DestinationURL, systemID, "TPI", false, appRoller)
		appReporter = NewAsyncReporter(config.DestinationURL, systemID, "Application", true, appRoller)
	}

	// TPI Writer Construction
	var tpiWriters []io.Writer
	tpiWriters = append(tpiWriters, tpiRoller)
	if tpiReporter != nil {
		tpiWriters = append(tpiWriters, tpiReporter)
	}

	if config.Verbose {
		tpiWriters = append(tpiWriters, os.Stdout)
	}
	tpiWriter := io.MultiWriter(tpiWriters...)
	tpiLogger := log.New(tpiWriter, "", 0) // flags=0 means NO timestamp, NO prefix

	// App Writer Construction
	var appWriters []io.Writer
	appWriters = append(appWriters, appRoller)
	if appReporter != nil {
		appWriters = append(appWriters, appReporter)
	}

	if config.Verbose {
		appWriters = append(appWriters, os.Stdout)
	}
	appWriter := io.MultiWriter(appWriters...)
	appLogger := log.New(appWriter, "", log.LstdFlags)

	return tpiLogger, appLogger, nil
}
