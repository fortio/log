// Copyright 2017-2023 Fortio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log // import "fortio.org/fortio/log"

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// leave this test first/where it is as it relies on line number not changing.
func TestLoggerFilenameLine(t *testing.T) {
	SetLogLevel(Debug) // make sure it's already debug when we capture
	on := true
	Config.LogFileAndLine = on
	Config.LogPrefix = "-prefix-"
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	SetFlags(0)
	SetLogLevel(Debug)
	if LogDebug() {
		Debugf("test") // line 41
	}
	w.Flush()
	actual := b.String()
	expected := "D logger_test.go:41-prefix-test\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func TestSetLevel(t *testing.T) {
	_ = SetLogLevel(Info)
	err := SetLogLevelStr("debug")
	if err != nil {
		t.Errorf("unexpected error for valid level %v", err)
	}
	prev := SetLogLevel(Info)
	if prev != Debug {
		t.Errorf("unexpected level after setting debug %v", prev)
	}
	err = SetLogLevelStr("bogus")
	if err == nil {
		t.Errorf("Didn't get an error setting bogus level")
	}
}

func TestLogger1(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(Info) // reset from other tests
	Config.LogFileAndLine = false
	Config.LogPrefix = ""
	log.SetOutput(w)
	log.SetFlags(0)
	// Start of the actual test
	SetLogLevel(LevelByName("Verbose"))
	expected := "I Log level is now 1 Verbose (was 2 Info)\n"
	i := 0
	if LogVerbose() {
		LogVf("test Va %d", i) // Should show
	}
	i++
	expected += "V test Va 0\n"
	Warnf("test Wa %d", i) // Should show
	i++
	expected += "W test Wa 1\n"
	Logger().Printf("test Logger().Printf %d", i)
	i++
	expected += "I test Logger().Printf 2\n"
	SetLogLevelQuiet(Debug)                        // no additional logging about level change
	prevLevel := SetLogLevel(LevelByName("error")) // works with lowercase too
	expected += "I Log level is now 4 Error (was 0 Debug)\n"
	LogVf("test Vb %d", i)                       // Should not show
	Infof("test info when level is error %d", i) // Should not show
	i++
	Warnf("test Wb %d", i) // Should not show
	i++
	Errf("test E %d", i) // Should show
	i++
	expected += "E test E 5\n"
	// test the rest of the api
	Logf(LevelByName("Critical"), "test %d level str %s, cur %s", i, prevLevel.String(), GetLogLevel().String())
	expected += "C test 6 level str Debug, cur Error\n"
	i++
	SetLogLevel(Debug) // should be fine and invisible change
	SetLogLevel(Debug - 1)
	expected += "SetLogLevel called with level -1 lower than Debug!\n"
	SetLogLevel(Fatal) // Hiding critical level is not allowed
	expected += "SetLogLevel called with level 6 higher than Critical!\n"
	SetLogLevel(Critical) // should be fine
	expected += "I Log level is now 5 Critical (was 0 Debug)\n"
	Critf("testing crit %d", i) // should show
	expected += "C testing crit 7\n"
	Printf("Printf should always show n=%d", 8)
	expected += "Printf should always show n=8\n"
	r := FErrf("FErrf should always show but not exit, n=%d", 9)
	expected += "F FErrf should always show but not exit, n=9\n"
	if r != 1 {
		t.Errorf("FErrf returned %d instead of 1", r)
	}
	_ = w.Flush()
	actual := b.String()
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func TestLogFatal(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic from log.Fatalf, didn't get one")
		}
	}()
	Fatalf("test of log fatal")
}

func TestLoggerFatalCliMode(t *testing.T) {
	SetDefaultsForClientTools()
	if os.Getenv("DO_LOG_FATALF") == "1" {
		Fatalf("test")
		Errf("should have exited / this shouldn't have been reached")
		return // will cause exit status 0 if reached and thus fail the test
	}
	// unfortunately, even if passing -test.coverprofile it doesn't get counted
	cmd := exec.Command(os.Args[0], "-test.run=TestLoggerFatalCliMode")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(), "DO_LOG_FATALF=1")
	err := cmd.Run()
	var e *exec.ExitError
	if ok := errors.As(err, &e); ok && e.ExitCode() == 1 {
		Printf("Got expected exit status 1")
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestLoggerFatalExitOverride(t *testing.T) {
	SetDefaultsForClientTools()
	exitCalled := false
	Config.FatalExit = func(code int) {
		exitCalled = true
	}
	Fatalf("testing log.Fatalf exit case")
	if !exitCalled {
		t.Error("expected exit function override not called")
	}
}

func TestMultipleFlags(t *testing.T) {
	SetLogLevelQuiet(Verbose)
	// use x... so it's sorted after the standard loglevel for package level
	// print default tests were all 3 flags are present.
	LoggerStaticFlagSetup("xllvl1", "xllvl2")
	f := flag.Lookup("loglevel")
	if f != nil {
		t.Error("expected default loglevel to not be registered")
	}
	f = flag.Lookup("xllvl1")
	if f.Value.String() != "Verbose" {
		t.Errorf("expected flag default value to be Verbose, got %s", f.Value.String())
	}
	if err := f.Value.Set("  iNFo\n"); err != nil {
		t.Errorf("expected flag to be settable, got %v", err)
	}
	f2 := flag.Lookup("xllvl2")
	if f2.Value.String() != "Info" {
		t.Errorf("expected linked flag value to be Info, got %s", f2.Value.String())
	}
	if GetLogLevel() != Info {
		t.Errorf("expected log level to be Info, got %s", GetLogLevel().String())
	}
	if err := f2.Value.Set("debug"); err != nil {
		t.Errorf("expected flag2 to be settable, got %v", err)
	}
	if GetLogLevel() != Debug {
		t.Errorf("expected log level to be Debug, got %s", GetLogLevel().String())
	}
}

func TestStaticFlagDefault(t *testing.T) {
	SetLogLevelQuiet(Warning)
	LoggerStaticFlagSetup()
	var b bytes.Buffer
	flag.CommandLine.SetOutput(&b)
	flag.CommandLine.PrintDefaults()
	s := b.String()
	expected := "  -loglevel level\n" +
		"    \tlog level, one of [Debug Verbose Info Warning Error Critical Fatal] " +
		"(default Warning)\n"
	if !strings.HasPrefix(s, expected) {
		t.Errorf("expected flag output to start with %q, got %q", expected, s)
	}
	f := flag.Lookup("loglevel")
	if f == nil {
		t.Fatal("expected flag to be registered")
	}
	if f.Value.String() != "Warning" {
		t.Errorf("expected flag default value to be Warning, got %s", f.Value.String())
	}
	if err := f.Value.Set("badlevel"); err == nil {
		t.Error("expected error passing a bad level value, didn't get one")
	}
	if err := f.Value.Set("  iNFo\n"); err != nil {
		t.Errorf("expected flag to be settable, got %v", err)
	}
	if GetLogLevel() != Info {
		t.Errorf("expected log level to be Info, got %s", GetLogLevel().String())
	}
}

func BenchmarkLogDirect1(b *testing.B) {
	setLevel(Error)
	for n := 0; n < b.N; n++ {
		Debugf("foo bar %d", n)
	}
}

func BenchmarkLogDirect2(b *testing.B) {
	setLevel(Error)
	for n := 0; n < b.N; n++ {
		Logf(Debug, "foo bar %d", n)
	}
}
