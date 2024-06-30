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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"fortio.org/log/goroutine"
)

const thisFilename = "logger_test.go"

// leave this test first/where it is as it relies on line number not changing.
func TestLoggerFilenameLine(t *testing.T) {
	SetLogLevel(Debug) // make sure it's already debug when we capture
	on := true
	Config.LogFileAndLine = on
	Config.LogPrefix = "-prefix-"
	Config.JSON = false
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	SetFlags(0)
	SetLogLevel(Debug)
	if LogDebug() {
		Debugf("test") // line 51
	}
	SetLogLevel(-1)      // line 53
	SetLogLevel(Warning) // line 54
	Infof("should not show (info level)")
	Printf("Should show despite being Info - unconditional Printf without line/file")
	w.Flush()
	actual := b.String()
	expected := "[D] logger_test.go:51-prefix-test\n" +
		"[E] logger_test.go:53-prefix-SetLogLevel called with level -1 lower than Debug!\n" +
		"[I] logger_test.go:54-prefix-Log level is now 3 Warning (was 0 Debug)\n" +
		"Should show despite being Info - unconditional Printf without line/file\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

// leave this test second/where it is as it relies on line number not changing.
func TestLoggerFilenameLineJSON(t *testing.T) {
	SetLogLevel(Debug) // make sure it's already debug when we capture
	on := true
	Config.LogFileAndLine = on
	Config.LogPrefix = "-not used-"
	Config.JSON = true
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	SetLogLevel(Debug)
	if LogDebug() {
		Debugf("a test") // line 81
	}
	w.Flush()
	actual := b.String()
	grID := goroutine.ID()
	if grID <= 0 {
		t.Errorf("unexpected goroutine id %d", grID)
	}
	expected := `{"level":"dbug","r":` + strconv.FormatInt(grID, 10) +
		`,"file":"` + thisFilename + `","line":81,"msg":"a test"}` + "\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func Test_LogS_JSON_no_json_with_filename(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Warning"))
	Config.LogFileAndLine = true
	Config.JSON = false
	Config.NoTimestamp = false
	Config.LogPrefix = "-bar-"
	log.SetFlags(0)
	SetOutput(w)
	// Start of the actual test
	S(Verbose, "This won't show")
	S(Warning, "This will show", Str("key1", "value 1"), Attr("key2", 42)) // line 109
	Printf("This will show too")                                           // no filename/line and shows despite level
	_ = w.Flush()
	actual := b.String()
	expected := "[W] logger_test.go:109-bar-This will show, key1=\"value 1\", key2=42\n" +
		"This will show too\n"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestColorMode(t *testing.T) {
	if ConsoleLogging() {
		t.Errorf("expected not to be console logging")
	}
	if Color {
		t.Errorf("expected to not be in color mode initially")
	}
	// Setup
	Config = DefaultConfig()
	Config.ForceColor = true
	Config.NoTimestamp = true
	Config.LogPrefix = "" // test it'll be at least one space
	SetLogLevelQuiet(Info)
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w) // will call SetColorMode()
	if !Color {
		t.Errorf("expected to be in color mode after ForceColor=true and SetColorMode()")
	}
	S(Warning, "With file and line", String("attr", "value with space")) // line 139
	Infof("info with file and line = %v", Config.LogFileAndLine)         // line 140
	Config.LogFileAndLine = false
	Config.GoroutineID = false
	S(Warning, "Without file and line", Str("attr", "value with space"))
	Infof("info with file and line = %v", Config.LogFileAndLine)
	_ = w.Flush()
	actual := b.String()
	grID := fmt.Sprintf("r%d ", goroutine.ID())
	expected := "\x1b[37m" + grID + "\x1b[90m[\x1b[33mWRN\x1b[90m] logger_test.go:139 " +
		"\x1b[33mWith file and line\x1b[0m, \x1b[34mattr\x1b[0m=\x1b[33m\"value with space\"\x1b[0m\n" +
		"\x1b[37m" + grID + "\x1b[90m[\x1b[32mINF\x1b[90m] logger_test.go:140 \x1b[32minfo with file and line = true\x1b[0m\n" +
		"\x1b[90m[\x1b[33mWRN\x1b[90m] \x1b[33mWithout file and line\x1b[0m, \x1b[34mattr\x1b[0m=\x1b[33m\"value with space\"\x1b[0m\n" +
		"\x1b[90m[\x1b[32mINF\x1b[90m] \x1b[32minfo with file and line = false\x1b[0m\n"
	if actual != expected {
		t.Errorf("got:\n%s\nexpected:\n%s", actual, expected)
	}
	// See color timestamp
	Config.NoTimestamp = false
	cs := colorTimestamp()
	if cs == "" {
		t.Errorf("got empty color timestamp")
	}
	if Colors.Green == "" {
		t.Errorf("expected to have green color not empty when in color mode")
	}
	prevGreen := Colors.Green
	// turn off color mode
	Config.ForceColor = false
	SetColorMode()
	if Color {
		t.Errorf("expected to not be in color mode after SetColorMode() and forcecolor false")
	}
	if Colors.Green != "" {
		t.Errorf("expected to have green color empty when not color mode, got %q", Colors.Green)
	}
	if LevelToColor[Info] != "" {
		t.Errorf("expected LevelToColor to be empty when not color mode, got %q", LevelToColor[Info])
	}
	// Show one can mutate/change/tweak colors
	customColor := "foo"
	ANSIColors.Green = customColor
	Config.ForceColor = true
	SetColorMode()
	if Colors.Green != customColor {
		t.Errorf("expected to have color customized, got %q", Colors.Green)
	}
	if LevelToColor[Info] != customColor {
		t.Errorf("expected LevelToColor to the custom foo, got %q", LevelToColor[Info])
	}
	// put it back to real green for other tests
	ANSIColors.Green = prevGreen
	// Reset for other/further tests
	Config.ForceColor = false
	SetColorMode()
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
	Config.JSON = false
	SetOutput(w)
	log.SetFlags(0)
	// Start of the actual test
	SetLogLevel(LevelByName("Verbose"))
	expected := "[I] Log level is now 1 Verbose (was 2 Info)\n"
	i := 0
	if LogVerbose() {
		LogVf("test Va %d", i) // Should show
	}
	i++
	expected += "[V] test Va 0\n"
	Warnf("test Wa %d", i) // Should show
	i++
	expected += "[W] test Wa 1\n"
	Logger().Printf("test Logger().Printf %d", i)
	i++
	expected += "[I] test Logger().Printf 2\n"
	SetLogLevelQuiet(Debug)                        // no additional logging about level change
	prevLevel := SetLogLevel(LevelByName("error")) // works with lowercase too
	expected += "[I] Log level is now 4 Error (was 0 Debug)\n"
	LogVf("test Vb %d", i)                       // Should not show
	Infof("test info when level is error %d", i) // Should not show
	i++
	Warnf("test Wb %d", i) // Should not show
	i++
	Errf("test E %d", i) // Should show
	i++
	expected += "[E] test E 5\n"
	// test the rest of the api
	Logf(LevelByName("Critical"), "test %d level str %s, cur %s", i, prevLevel.String(), GetLogLevel().String())
	expected += "[C] test 6 level str Debug, cur Error\n"
	i++
	SetLogLevel(Debug) // should be fine and invisible change
	SetLogLevel(Debug - 1)
	expected += "[E] SetLogLevel called with level -1 lower than Debug!\n"
	SetLogLevel(Fatal) // Hiding critical level is not allowed
	expected += "[E] SetLogLevel called with level 6 higher than Critical!\n"
	SetLogLevel(Critical) // should be fine
	expected += "[I] Log level is now 5 Critical (was 0 Debug)\n"
	Critf("testing crit %d", i) // should show
	expected += "[C] testing crit 7\n"
	Printf("Printf should always show n=%d", 8)
	expected += "Printf should always show n=8\n"
	r := FErrf("FErrf should always show but not exit, n=%d", 9)
	expected += "[F] FErrf should always show but not exit, n=9\n"
	if r != 1 {
		t.Errorf("FErrf returned %d instead of 1", r)
	}
	_ = w.Flush()
	actual := b.String()
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func TestLoggerJSON(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = true
	Config.LogPrefix = "not used"
	Config.JSON = true
	Config.NoTimestamp = false
	SetOutput(w)
	// Start of the actual test
	now := time.Now()
	if LogVerbose() {
		LogVf("Test Verbose %d", 0) // Should show
	}
	_ = w.Flush()
	actual := b.String()
	e := JSONEntry{}
	err := json.Unmarshal([]byte(actual), &e)
	t.Logf("got: %s -> %#v", actual, e)
	if err != nil {
		t.Errorf("unexpected JSON deserialization error %v for %q", err, actual)
	}
	if e.Level != "trace" {
		t.Errorf("unexpected level %s", e.Level)
	}
	if e.Msg != "Test Verbose 0" {
		t.Errorf("unexpected body %s", e.Msg)
	}
	if e.File != thisFilename {
		t.Errorf("unexpected file %q", e.File)
	}
	if e.Line < 270 || e.Line > 310 {
		t.Errorf("unexpected line %d", e.Line)
	}
	ts := e.Time()
	now = microsecondResolution(now) // truncates so can't be after ts
	if now.After(ts) {
		t.Errorf("unexpected time %v is after %v", now, ts)
	}
	if ts.Sub(now) > 100*time.Millisecond {
		t.Errorf("unexpected time %v is > 1sec after %v", ts, now)
	}
}

func Test_LogS_JSON(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = true
	Config.JSON = true
	Config.NoTimestamp = false
	SetOutput(w)
	// Start of the actual test
	now := time.Now()
	value2 := 42
	value3 := 3.14
	S(Verbose, "Test Verbose", Str("key1", "value 1"), Int("key2", value2), Float64("key3", value3))
	_ = w.Flush()
	actual := b.String()
	e := JSONEntry{}
	err := json.Unmarshal([]byte(actual), &e)
	t.Logf("got: %s -> %#v", actual, e)
	if err != nil {
		t.Errorf("unexpected JSON deserialization error %v for %q", err, actual)
	}
	if e.Level != "trace" {
		t.Errorf("unexpected level %s", e.Level)
	}
	if e.Msg != "Test Verbose" {
		t.Errorf("unexpected body %s", e.Msg)
	}
	if e.File != thisFilename {
		t.Errorf("unexpected file %q", e.File)
	}
	if e.Line < 270 || e.Line > 340 {
		t.Errorf("unexpected line %d", e.Line)
	}
	ts := e.Time()
	now = microsecondResolution(now) // truncates so can't be after ts
	if now.After(ts) {
		t.Errorf("unexpected time %v is after %v", now, ts)
	}
	if ts.Sub(now) > 100*time.Millisecond {
		t.Errorf("unexpected time %v is > 1sec after %v", ts, now)
	}
	// check extra attributes
	var tmp map[string]interface{}
	err = json.Unmarshal([]byte(actual), &tmp)
	if err != nil {
		t.Errorf("unexpected JSON deserialization 2 error %v for %q", err, actual)
	}
	if tmp["key1"] != "value 1" {
		t.Errorf("unexpected key1 %v", tmp["key1"])
	}
	if tmp["key2"] != float64(42) {
		t.Errorf("unexpected key2 %v", tmp["key2"])
	}
	if tmp["key3"] != 3.14 { // comparing floats with == is dicey but... this passes...
		t.Errorf("unexpected key3 %v", tmp["key3"])
	}
	if tmp["file"] != thisFilename {
		t.Errorf("unexpected file %v", tmp["file"])
	}
}

func Test_LogS_JSON_no_file(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Warning"))
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = false
	SetOutput(w)
	// Start of the actual test
	S(Verbose, "This won't show")
	S(Warning, "This will show", Attr("key1", "value 1"))
	_ = w.Flush()
	actual := b.String()
	var tmp map[string]interface{}
	err := json.Unmarshal([]byte(actual), &tmp)
	if err != nil {
		t.Errorf("unexpected JSON deserialization error %v for %q", err, actual)
	}
	if tmp["key1"] != "value 1" {
		t.Errorf("unexpected key1 %v", tmp["key1"])
	}
	if tmp["file"] != nil {
		t.Errorf("unexpected file %v", tmp["file"])
	}
}

func Test_LogS_JSON_no_json_no_file(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Warning"))
	Config.LogFileAndLine = false
	Config.JSON = false
	Config.NoTimestamp = false
	Config.LogPrefix = "-foo-"
	log.SetFlags(0)
	SetOutput(w)
	// Start of the actual test
	S(Verbose, "This won't show")
	S(Warning, "This will show", Str("key1", "value 1"), Attr("key2", 42))
	S(NoLevel, "This NoLevel will show despite logically info level")
	_ = w.Flush()
	actual := b.String()
	expected := "[W]-foo-This will show, key1=\"value 1\", key2=42\n" +
		"This NoLevel will show despite logically info level\n"
	if actual != expected {
		t.Errorf("---got:---\n%s\n---expected:---\n%s\n", actual, expected)
	}
}

func TestLoggerJSONNoTimestampNoFilename(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = false
	Config.LogPrefix = "no used"
	Config.JSON = true
	Config.NoTimestamp = true
	SetOutput(w)
	// Start of the actual test
	Critf("Test Critf")
	_ = w.Flush()
	actual := b.String()
	e := JSONEntry{}
	err := json.Unmarshal([]byte(actual), &e)
	t.Logf("got: %s -> %#v", actual, e)
	if err != nil {
		t.Errorf("unexpected JSON deserialization error %v for %q", err, actual)
	}
	if e.Level != "crit" {
		t.Errorf("unexpected level %s", e.Level)
	}
	if e.Msg != "Test Critf" {
		t.Errorf("unexpected body %s", e.Msg)
	}
	if e.File != "" {
		t.Errorf("unexpected file %q", e.File)
	}
	if e.Line != 0 {
		t.Errorf("unexpected line %d", e.Line)
	}
	if e.TS != 0 {
		t.Errorf("unexpected time should be absent, got %v %v", e.TS, e.Time())
	}
}

func TestLoggerSimpleJSON(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = false
	Config.LogPrefix = "no used"
	Config.JSON = true
	Config.NoTimestamp = false
	SetOutput(w)
	// Start of the actual test
	w.WriteString("[")
	Critf("Test Critf2")
	w.WriteString(",")
	S(Critical, "Test Critf3")
	w.WriteString("]")
	_ = w.Flush()
	actual := b.String()
	e := []JSONEntry{}
	err := json.Unmarshal([]byte(actual), &e)
	t.Logf("got: %s -> %#v", actual, e)
	if err != nil {
		t.Errorf("unexpected JSON deserialization error %v for %q", err, actual)
	}
	if len(e) != 2 {
		t.Errorf("unexpected number of entries %d", len(e))
	}
	for i := 0; i < 2; i++ {
		e := e[i]
		if e.Level != "crit" {
			t.Errorf("unexpected level %s", e.Level)
		}
		exp := fmt.Sprintf("Test Critf%d", i+2)
		if e.Msg != exp {
			t.Errorf("unexpected body %s", e.Msg)
		}
		if e.File != "" {
			t.Errorf("unexpected file %q", e.File)
		}
		if e.Line != 0 {
			t.Errorf("unexpected line %d", e.Line)
		}
		if e.TS == 0 {
			t.Errorf("unexpected 0 time should have been present")
		}
	}
}

// Test that TimeToTs and Time() are inverse of one another.
func TestTimeToTs(t *testing.T) {
	var prev float64
	// tight loop to get different times, at highest resolution
	for i := 0; i < 100000; i++ {
		now := time.Now()
		// now = now.Add(69 * time.Nanosecond)
		usecTSstr := timeToTStr(now)
		usecTS, _ := strconv.ParseFloat(usecTSstr, 64)
		if i != 0 && usecTS < prev {
			t.Fatalf("clock went back in time at iter %d %v vs %v", i, usecTS, prev)
		}
		prev = usecTS
		e := JSONEntry{TS: usecTS}
		inv := e.Time()
		// Round to microsecond because that's the resolution of the timestamp
		// (note that on a mac for instance, there is no nanosecond resolution anyway)
		if !microsecondResolution(now).Equal(inv) {
			t.Fatalf("[at %d] unexpected time %v -> %v != %v (%v %v)", i, now, microsecondResolution(now), inv, usecTS, usecTSstr)
		}
	}
}

func microsecondResolution(t time.Time) time.Time {
	// Truncate and not Round because that's what UnixMicro does (indirectly).
	return t.Truncate(1 * time.Microsecond)
}

// concurrency test, make sure json aren't mixed up.
func TestLoggerJSONConcurrency(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = true
	Config.NoTimestamp = true
	Config.JSON = true
	SetOutput(w)
	// Start of the actual test
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			for j := 0; j < 100; j++ {
				Infof("Test from %d: %d", i, j)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	_ = w.Flush()
	actual := b.String()
	t.Logf("got: %s", actual)
	// Check it all deserializes to JSON correctly and we get the expected number of lines
	count := 0
	for _, line := range strings.Split(actual, "\n") {
		if count == 1000 && line == "" {
			// last line is empty
			continue
		}
		count++
		e := JSONEntry{}
		err := json.Unmarshal([]byte(line), &e)
		if err != nil {
			t.Errorf("unexpected JSON deserialization error on line %d %v for %q", count, err, line)
		}
	}
	if count != 1000 {
		t.Errorf("unexpected number of lines %d", count)
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
	Config.FatalExit = func(_ int) {
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

func TestTimeToTS(t *testing.T) {
	// test a few times and expected string
	for _, tst := range []struct {
		sec    int64
		nano   int64
		result string
	}{
		{1688763601, 42000, "1688763601.000042"},     // 42 usec after the seconds part, checking for leading zeroes
		{1688763601, 199999999, "1688763601.199999"}, // nanosec are truncated away not rounded (see note in TimeToTS)
		{1688763601, 200000999, "1688763601.200000"}, // boundary
		{1689983019, 142600000, "1689983019.142600"}, // trailing zeroes
	} {
		tm := time.Unix(tst.sec, tst.nano)
		ts := timeToTStr(tm)
		if ts != tst.result {
			t.Errorf("unexpected ts for %d, %d -> %q instead of %q (%v)", tst.sec, tst.nano, ts, tst.result, tm)
		}
	}
}

func TestJSONLevelReverse(t *testing.T) {
	str := LevelToJSON[Warning]
	if str != `"warn"` {
		t.Errorf("unexpected JSON level string %q (should have quotes)", str)
	}
	lvl := JSONStringLevelToLevel["warn"]
	if lvl != Warning {
		t.Errorf("unexpected level %d", lvl)
	}
	lvl = JSONStringLevelToLevel["info"] // Should be info and not NoLevel (7)
	if lvl != Info {
		t.Errorf("unexpected level %d", lvl)
	}
	lvl = JSONStringLevelToLevel["fatal"] // Should be info and not NoLevel (7)
	if lvl != Fatal {
		t.Errorf("unexpected level %d", lvl)
	}
}

func TestNoLevel(t *testing.T) {
	Config.ForceColor = true
	SetColorMode()
	color := ColorLevelToStr(NoLevel)
	if color != ANSIColors.DarkGray {
		t.Errorf("unexpected color %q", color)
	}
	Config.ForceColor = false
	Config.JSON = true
	Config.ConsoleColor = false
	Config.NoTimestamp = true
	Config.GoroutineID = false
	var buf bytes.Buffer
	SetOutput(&buf)
	Printf("test")
	actual := buf.String()
	expected := `{"level":"info","msg":"test"}` + "\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

type customError struct {
	Msg  string
	Code int
}

type customErrorAlias customError

func (e customError) Error() string {
	return fmt.Sprintf("custom error %s (code %d)", e.Msg, e.Code)
}

func (e customError) MarshalJSON() ([]byte, error) {
	return json.Marshal(customErrorAlias(e))
}

func TestPointers(t *testing.T) {
	var iPtr *int
	kv := Any("err", iPtr)
	kvStr := kv.StringValue()
	expected := `null`
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
	i := 42
	iPtr = &i
	kv = Any("int", iPtr)
	kvStr = kv.StringValue()
	expected = `42`
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
	var sPtr *string
	kv = Any("msg", sPtr)
	kvStr = kv.StringValue()
	expected = `null`
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
	s := "test\nline2"
	sPtr = &s
	kv = Any("msg", sPtr)
	kvStr = kv.StringValue()
	expected = `"test\nline2"`
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
}

func TestStruct(t *testing.T) {
	type testStruct struct {
		Msg1 string
		Msg2 *string
	}
	ptrStr := "test2"
	ts := testStruct{Msg1: "test\nline2", Msg2: &ptrStr}
	kv := Any("ts", ts)
	kvStr := kv.StringValue()
	expected := `{"Msg1":"test\nline2","Msg2":"test2"}`
	if !fullJSON {
		expected = `"{Msg1:test\nline2 Msg2:`
		expected += fmt.Sprintf("%p}\"", &ptrStr)
	}
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
}

func TestSerializationOfError(t *testing.T) {
	var err error
	kv := Any("err", err)
	kvStr := kv.StringValue()
	expected := `null`
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
	err = errors.New("test error")
	Errf("Error on purpose: %v", err)
	S(Error, "Error on purpose", Any("err", err))
	kv = Any("err", err)
	kvStr = kv.StringValue()
	expected = `"test error"`
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
	err = customError{Msg: "custom error", Code: 42}
	kv = Any("err", err)
	kvStr = kv.StringValue()
	expected = `{"Msg":"custom error","Code":42}`
	if !fullJSON {
		expected = `"custom error custom error (code 42)"`
	}
	if kvStr != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", kvStr, expected)
	}
}

func TestEnvHelp(t *testing.T) {
	SetDefaultsForClientTools()
	Config.NoTimestamp = false
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	EnvHelp(w)
	w.Flush()
	actual := b.String()
	expected := `# Logger environment variables:
LOGGER_LOG_PREFIX=' '
LOGGER_LOG_FILE_AND_LINE=false
LOGGER_FATAL_PANICS=false
LOGGER_JSON=false
LOGGER_NO_TIMESTAMP=false
LOGGER_CONSOLE_COLOR=true
LOGGER_FORCE_COLOR=false
LOGGER_GOROUTINE_ID=false
LOGGER_COMBINE_REQUEST_AND_RESPONSE=false
LOGGER_LEVEL='Info'
`
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func TestConfigFromEnvError(t *testing.T) {
	t.Setenv("LOGGER_LEVEL", "foo")
	var buf bytes.Buffer
	SetOutput(&buf)
	configFromEnv()
	actual := buf.String()
	expected := "Invalid log level from environment"
	if !strings.Contains(actual, expected) {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func TestConfigFromEnvOk(t *testing.T) {
	t.Setenv("LOGGER_LEVEL", "verbose")
	var buf bytes.Buffer
	SetOutput(&buf)
	configFromEnv()
	actual := buf.String()
	expected := "Log level set from environment LOGGER_LEVEL to Verbose"
	if !strings.Contains(actual, expected) {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

// io.Discard but specially known to by logger optimizations for instance.
type discard struct{}

func (discard) Write(p []byte) (int, error) {
	return len(p), nil
}

func (discard) WriteString(s string) (int, error) {
	return len(s), nil
}

var Discard = discard{}

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

func BenchmarkMultipleStrNoLog(b *testing.B) {
	setLevel(Error)
	for n := 0; n < b.N; n++ {
		S(Debug, "foo bar", Str("a", "aval"), Str("b", "bval"), Str("c", "cval"), Str("d", "dval"))
	}
}

func BenchmarkLogSnologNotOptimized1(b *testing.B) {
	for n := 0; n < b.N; n++ {
		// Avoid optimization for n < 256 that skews memory number (and combined with truncation gives 0 instead of 1)
		// https://github.com/golang/go/blob/go1.21.0/src/runtime/iface.go#L493
		S(Debug, "foo bar", Attr("n1", 12345+n))
	}
}

func BenchmarkLogSnologNotOptimized4(b *testing.B) {
	for n := 0; n < b.N; n++ {
		v := n + 12345
		S(Debug, "foo bar", Attr("n1", v), Attr("n2", v+1), Attr("n3", v+2), Attr("n4", v+3))
	}
}

func BenchmarkLogSnologOptimized(b *testing.B) {
	setLevel(Error)
	v := ValueType[int]{0}
	aa := KeyVal{Key: "n", Value: &v}
	ba := Str("b", "bval")
	for n := 0; n < b.N; n++ {
		v.Val = n + 1235
		S(Debug, "foo bar", aa, ba)
	}
}

func BenchmarkLogS_NotOptimized(b *testing.B) {
	setLevel(Info)
	Config.JSON = true
	Config.LogFileAndLine = false
	Config.ConsoleColor = false
	Config.ForceColor = false
	SetOutput(Discard)
	for n := 0; n < b.N; n++ {
		S(Info, "foo bar", Attr("n", n))
	}
}

func BenchmarkLog_Optimized(b *testing.B) {
	setLevel(Info)
	Config.JSON = true
	Config.LogFileAndLine = false
	Config.ConsoleColor = false
	Config.ForceColor = false
	SetOutput(Discard)
	v := ValueType[int]{0}
	a := KeyVal{Key: "n", Value: &v}
	for n := 0; n < b.N; n++ {
		v.Val = n
		S(Info, "foo bar", a)
	}
}

func BenchmarkLogOldStyle(b *testing.B) {
	setLevel(Info)
	Config.JSON = false
	Config.LogFileAndLine = false
	Config.ConsoleColor = false
	Config.ForceColor = false
	SetColorMode()
	SetOutput(Discard)
	for n := 0; n < b.N; n++ {
		S(Info, "foo bar", Attr("n", n))
	}
}
