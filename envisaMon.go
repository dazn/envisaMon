package main

import (
	"envisaMon/tpi"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// 1. Parse command-line arguments and flags
	ipAddress, port, printMessages, printAppLog, deduplicate := parseArgs()

	// 2. Read password from environment
	password := os.Getenv("ENVISALINK_TPI_KEY")
	if password == "" {
		fmt.Fprintln(os.Stderr, "ERROR: ENVISALINK_TPI_KEY environment variable not set")
		os.Exit(1)
	}

	// 3. Set up dual logging with lumberjack
	tpiLogger, appLogger := setupLogging(printMessages, printAppLog)

	// 4. Create TPI client
	client := tpi.NewClient(
		fmt.Sprintf("%s:%d", ipAddress, port),
		password,
		tpiLogger,
		appLogger,
		deduplicate,
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
	appLogger.Printf("INFO: Starting TPI monitor for %s", ipAddress)
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

func parseArgs() (string, int, bool, bool, bool) {
	printMessages := flag.Bool("m", false, "print TPI messages to stdout")
	printAppLog := flag.Bool("l", false, "print application log to stdout")
	deduplicate := flag.Bool("u", false, "deduplicate consecutive identical TPI messages")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <ip-address> [port]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -m    print TPI messages to stdout (in addition to log file)\n")
		fmt.Fprintf(os.Stderr, "  -l    print application log to stdout (in addition to log file)\n")
		fmt.Fprintf(os.Stderr, "  -u    deduplicate consecutive identical TPI messages\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s 192.168.1.100              # Uses default port 4025\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 192.168.1.100 2222         # Uses custom port 2222\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -m -l 192.168.1.100 2222   # With logging options\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	ipAddress := flag.Arg(0)
	port := 4025 // default port

	// Parse optional port argument
	if flag.NArg() >= 2 {
		var err error
		port, err = strconv.Atoi(flag.Arg(1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Invalid port number: %s\n", flag.Arg(1))
			os.Exit(1)
		}
		if port < 1 || port > 65535 {
			fmt.Fprintf(os.Stderr, "ERROR: Port must be between 1 and 65535, got: %d\n", port)
			os.Exit(1)
		}
	}

	return ipAddress, port, *printMessages, *printAppLog, *deduplicate
}

func setupLogging(printMessages, printAppLog bool) (*log.Logger, *log.Logger) {
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

	var tpiWriter io.Writer = tpiRoller
	if printMessages {
		tpiWriter = io.MultiWriter(tpiRoller, os.Stdout)
	}
	tpiLogger := log.New(tpiWriter, "", 0) // flags=0 means NO timestamp, NO prefix

	// Application event logger
	appRoller := &lumberjack.Logger{
		Filename:   "./logs/application.log",
		MaxSize:    5, // megabytes
		MaxBackups: 3, // keep only the 3 most recent log files
		Compress:   true,
	}

	var appWriter io.Writer = appRoller
	if printAppLog {
		appWriter = io.MultiWriter(appRoller, os.Stdout)
	}
	appLogger := log.New(appWriter, "", log.LstdFlags)

	return tpiLogger, appLogger
}
