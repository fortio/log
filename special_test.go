package log

import (
	"bufio"
	"bytes"
	"math"
	"testing"
)

func Test_LogS_JSON_NaN(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	SetOutput(w)
	// Start of the actual test
	value := math.NaN()
	zero := 0.0
	S(Verbose, "Test NaN", Any("nan", value), Any("minus-inf", -1.0/zero))
	_ = w.Flush()
	actual := b.String()
	// Note that we serialize that way but can't deserialize with go default json unmarshaller
	expected := `{"level":"trace","msg":"Test NaN","nan":NaN,"minus-inf":-Inf}` + "\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func Test_LogS_JSON_Array(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	SetOutput(w)
	// Start of the actual test
	S(Verbose, "Test Array", Any("arr", []interface{}{"x", 42, "y"}))
	_ = w.Flush()
	actual := b.String()
	expected := `{"level":"trace","msg":"Test Array","arr":["x",42,"y"]}` + "\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func Test_LogS_JSON_Map(t *testing.T) {
	// Setup
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetLogLevel(LevelByName("Verbose"))
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	SetOutput(w)
	// Start of the actual test
	tst := map[string]interface{}{
		"str1": "val 1",
		"subArray": []interface{}{
			"x", 42, "y",
		},
		"number": 3.14,
	}
	S(Verbose, "Test Map", Any("map", tst), Int64("in64", 0), Bool("bool", true))
	_ = w.Flush()
	actual := b.String()
	//nolint:lll
	expected := `{"level":"trace","msg":"Test Map","map":{"number":3.14,"str1":"val 1","subArray":["x",42,"y"]},"in64":0,"bool":true}` +
		"\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}
