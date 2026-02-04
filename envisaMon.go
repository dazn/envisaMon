package main

import (
	"envisaMon/tpi"
	"flag"
	"fmt"
	"io"
	"log"
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
	config := parseArgs()

	// 2. Read password from environment
	password := os.Getenv("ENVISALINK_TPI_KEY")
	if password == "" {
		fmt.Fprintln(os.Stderr, "ERROR: ENVISALINK_TPI_KEY environment variable not set")
		os.Exit(1)
	}

	// 3. Set up dual logging with lumberjack
	tpiLogger, appLogger := setupLogging(config)
	appLogger.Printf("DEBUG: Parsed URL from arg: %s (path: %s)", config.DestinationURL, config.DestinationPath)

	// 4. Create TPI client
	client := tpi.NewClient(
		fmt.Sprintf("%s:%d", config.EnvisaLinkIP, config.EnvisaLinkPort),
		password,
		tpiLogger,
		appLogger,
		config.Deduplicate,
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
	PrintTPIMessages bool
	PrintAppLog      bool
	Deduplicate      bool
}

func parseArgs() *Config {
	config := &Config{}
	flag.BoolVar(&config.PrintTPIMessages, "m", false, "print TPI messages to stdout")
	flag.BoolVar(&config.PrintAppLog, "l", false, "print application log to stdout")
	flag.BoolVar(&config.Deduplicate, "u", false, "deduplicate consecutive identical TPI messages")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <ip>[:port] [<url>]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "  <ip>[:port]    EnvisaLink IP address, optionally with port (default: 4025)\n")
		fmt.Fprintf(os.Stderr, "  [<url>]        Optional: Event destination URL in format https://host[:port][/path/to/endpoint]\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -m    print TPI messages to stdout (in addition to log file)\n")
		fmt.Fprintf(os.Stderr, "  -l    print application log to stdout (in addition to log file)\n")
		fmt.Fprintf(os.Stderr, "  -u    deduplicate consecutive identical TPI messages\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s 192.168.1.100\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 192.168.1.100 https://events.example.com:8080\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 192.168.1.100:4026 https://events.example.com/webhook\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -m -l 192.168.1.100:4026 https://events.example.com:8080/api/events\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() < 1 || flag.NArg() > 2 {
		fmt.Fprintf(os.Stderr, "ERROR: Expected 1 or 2 arguments, got %d\n", flag.NArg())
		flag.Usage()
		os.Exit(1)
	}

	// Parse first argument: <ip>[:port]
	ipPortArg := flag.Arg(0)
	config.EnvisaLinkPort = 4025 // default port

	if strings.Contains(ipPortArg, ":") {
		parts := strings.SplitN(ipPortArg, ":", 2)
		config.EnvisaLinkIP = parts[0]

		var err error
		config.EnvisaLinkPort, err = strconv.Atoi(parts[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Invalid port number in '%s': %s\n", ipPortArg, parts[1])
			os.Exit(1)
		}
		if config.EnvisaLinkPort < 1 || config.EnvisaLinkPort > 65535 {
			fmt.Fprintf(os.Stderr, "ERROR: Port must be between 1 and 65535, got: %d\n", config.EnvisaLinkPort)
			os.Exit(1)
		}
	} else {
		config.EnvisaLinkIP = ipPortArg
	}

	// Parse second argument if present: URL in format https://host:port/path
	if flag.NArg() == 2 {
		urlArg := flag.Arg(1)
		config.DestinationURL = urlArg
		parsedURL, err := url.Parse(urlArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Invalid URL format: %v\n", err)
			os.Exit(1)
		}

		// Validate URL scheme
		if parsedURL.Scheme != "https" {
			fmt.Fprintf(os.Stderr, "ERROR: URL scheme must be 'https', got: '%s'\n", parsedURL.Scheme)
			os.Exit(1)
		}

		// Validate URL host is present
		if parsedURL.Host == "" {
			fmt.Fprintf(os.Stderr, "ERROR: URL must include a host\n")
			os.Exit(1)
		}

		// Validate URL port if present
		urlPort := parsedURL.Port()
		if urlPort != "" {
			urlPortNum, err := strconv.Atoi(urlPort)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: Invalid port in URL: %s\n", urlPort)
				os.Exit(1)
			}
			if urlPortNum < 1 || urlPortNum > 65535 {
				fmt.Fprintf(os.Stderr, "ERROR: URL port must be between 1 and 65535, got: %d\n", urlPortNum)
				os.Exit(1)
			}
		}

		// Extract the URL path (empty string if not present)
		config.DestinationPath = parsedURL.Path
	}

	return config
}

func setupLogging(config *Config) (*log.Logger, *log.Logger) {
	// Ensure logs directory exists
	if err := os.MkdirAll("./logs", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create logs directory: %v\n", err)
		os.Exit(1)
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
	systemID := config.EnvisaLinkIP
	if config.EnvisaLinkPort != 4025 {
		systemID = fmt.Sprintf("%s:%d", config.EnvisaLinkIP, config.EnvisaLinkPort)
	} else {
		systemID = fmt.Sprintf("%s:%d", config.EnvisaLinkIP, config.EnvisaLinkPort)
	}

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

	if config.PrintTPIMessages {
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

	if config.PrintAppLog {
		appWriters = append(appWriters, os.Stdout)
	}
	appWriter := io.MultiWriter(appWriters...)
	appLogger := log.New(appWriter, "", log.LstdFlags)

	return tpiLogger, appLogger
}
