// Initially from https://github.com/fortio/logc/blob/v1.1.0/levelsDemo/levels.go
package main

import (
	"fmt"

	"fortio.org/log"
)

func main() {
	log.SetLogLevelQuiet(log.Debug)
	// Could also call: log.SetDefaultsForClientTools() instead of setting these 3 fields manually.
	// but it would be ignored when redirecting.
	log.Config.LogFileAndLine = false
	log.Config.LogPrefix = ""
	log.Config.GoroutineID = false
	log.Debugf("This is a debug message without goroutine id, file:line nor prefix (cli style)")
	log.Config = log.DefaultConfig()
	// So log fatal doesn't panic nor exit (so we can print the non-JSON last line).
	log.Config.FatalPanics = false
	log.Config.FatalExit = func(int) {}
	// Meat of the example: (some of these are reproducing fixed issues in `logc` JSON->console attributes detection)
	log.Debugf("Back to default (server) logging style with a debug message ending with backslash \\")
	log.LogVf("This is a verbose message")
	log.Printf("This an always printed, file:line omitted message (and no level in console)")
	log.Infof("This is an info message with no attributes but with \"quotes\"...")
	log.S(log.Info, "This is multi line\n\tstructured info message with 3 attributes",
		log.Str("attr1", "value1"), log.Attr("attr2", 42), log.Str("attr3", "\"quoted\nvalue\""))
	log.Warnf("This is a warning message")
	log.Errf("This is an error message")
	log.Critf("This is a critical message")
	log.Fatalf("This is a fatal message") //nolint:revive // we disabled exit for this demo
	fmt.Println("This is a non-JSON output, will get prefixed with a exclamation point with logc")
}
