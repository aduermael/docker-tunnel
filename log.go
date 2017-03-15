package main

import (
	"fmt"
	"os"
)

const (
	logLevelDebug int = iota
	logLevelError
	logLevelInfo
)

var (
	logLevel = logLevelInfo
)

func print(args ...interface{}) {
	if logLevel <= logLevelInfo {
		fmt.Println(args...)
	}
}

func printError(args ...interface{}) {
	if logLevel <= logLevelError {
		fmt.Fprintln(os.Stderr, args...)
	}
}

func printFatal(args ...interface{}) {
	printError(args...)
	os.Exit(1)
}

func printDebug(args ...interface{}) {
	if logLevel <= logLevelDebug {
		fmt.Println(args...)
	}
}
