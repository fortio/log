// Copyright 2023 Fortio Authors
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

package log

import (
	"os"
	"time"
)

const (
	// ANSI color codes.
	reset     = "\033[0m"
	red       = "\033[31m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	white     = "\033[97m"
	brightRed = "\033[91m"
	darkGray  = "\033[90m"
)

var (
	// Mapping of log levels to color.
	LevelToColor = []string{
		gray,
		cyan,
		green,
		yellow,
		red,
		purple,
		brightRed,
	}
	// Cached flag for whether to use color output or not.
	Color = false
)

// ConsoleLogging is a utility to check if the current logger output is a console (terminal).
func ConsoleLogging() bool {
	f, ok := jsonWriter.(*os.File)
	if !ok {
		return false
	}
	s, _ := f.Stat()
	return (s.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// SetColorMode computes whether we currently should be using color text mode or not.
// Need to be reset if config changes (but is already automatically re evaluated when calling SetOutput()).
func SetColorMode() {
	Color = ColorMode()
}

// ColorMode returns true if we should be using color text mode, which is either because it's
// forced or because we are in a console and the config allows it.
// Should not be called often, instead read/update the Color variable when needed.
func ColorMode() bool {
	return Config.ForceColor || (Config.ConsoleColor && ConsoleLogging())
}

func colorTimestamp() string {
	if Config.NoTimestamp {
		return ""
	}
	return time.Now().Format("\033[90m15:04:05.000 ")
}
