package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

type testConfig struct {
	args []string
	err  error
	config
}

var binaryName string

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		binaryName = "application-test.exe"
	} else {
		binaryName = "application-test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// build the app:
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryName)
	err := cmd.Run()

	if err != nil {
		os.Exit(1)
	}

	// cleanup the test binary:
	defer func() {
		err = os.Remove(binaryName)
		if err != nil {
			log.Fatalf("Error removing built binary: %v", err)
		}
	}()
	m.Run()
}

func TestParseArgs(t *testing.T) {
	tests := []testConfig{
		{
			args:   []string{"-h"},
			err:    nil,
			config: config{printUsage: true, numTimes: 0},
		},
		{
			args:   []string{"10"},
			err:    nil,
			config: config{printUsage: false, numTimes: 10},
		},
		{
			args:   []string{"abc"},
			err:    errors.New("strconv.Atoi: parsing \"abc\": invalid syntax"),
			config: config{printUsage: false, numTimes: 0},
		},
		{
			args:   []string{"1", "foo"},
			err:    errors.New("invalid number of arguments"),
			config: config{printUsage: false, numTimes: 0},
		},
	}

	for _, tc := range tests {
		c, err := parseArgs(tc.args)
		if tc.err != nil && err.Error() != tc.err.Error() {
			t.Fatalf("expected error to be: %v, got: %v\n", tc.err, err)
		}
		if tc.err == nil && err != nil {
			t.Errorf("expected nil error, got: %v\n", err)
		}
		if c.printUsage != tc.printUsage {
			t.Errorf("expected printUsage to be: %v, got: %v\n", tc.printUsage, c.printUsage)
		}
		if c.numTimes != tc.numTimes {
			t.Errorf("expected numTimes to be: %v, got: %v\n", tc.numTimes, c.numTimes)
		}
	}
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		c   config
		err error
	}{
		{
			c:   config{},
			err: errors.New("must specify a number greater than 0"),
		},
		{
			c:   config{numTimes: -1},
			err: errors.New("must specify a number greater than 0"),
		},
		{
			c:   config{numTimes: 10},
			err: nil,
		},
	}

	for _, tc := range tests {
		err := validateArgs(tc.c)
		if tc.err != nil && err.Error() != tc.err.Error() {
			t.Errorf("expectetd error to be: %v, got: %v\n", tc.err, err)
		}
		if tc.err == nil && err != nil {
			t.Errorf("expected nil error, got: %v\n", err)
		}
	}
}

func TestRunCmd(t *testing.T) {
	tests := []struct {
		c      config
		input  string
		output string
		err    error
	}{
		{
			c:      config{printUsage: true},
			output: usageString,
		},
		{
			c:      config{numTimes: 5},
			input:  "",
			output: strings.Repeat("Your name please? Press the return key when done.\n", 1),
			err:    errors.New("you didn't enter your name"),
		},
		{
			c:      config{numTimes: 5},
			input:  "Benny Engstrom",
			output: "Your name please? Press the return key when done.\n" + strings.Repeat("Nice to meet you Benny Engstrom\n", 5),
		},
	}

	// To mimic the standard output, we create an empty Buffer object that implements the `Writer` interface using `new(bytes.Buffer)`
	byteBuf := new(bytes.Buffer)

	for _, tc := range tests {
		// To mimc an input from the user, this is how you can create an `io.Reader` from a string:
		rd := strings.NewReader(tc.input)
		// When the getName() function is called with `io.Reader r` scanner.Text() will return the string in tc.input

		err := runCmd(rd, byteBuf, tc.c)

		if err != nil && tc.err == nil {
			t.Fatalf("expected nil error, got: %v\n", err)
		}
		if tc.err != nil && err.Error() != tc.err.Error() {
			t.Fatalf("expected error: %v, got error: %v\n", tc.err.Error(), err.Error())
		}

		// `byteBuf.String()` allows us to obtain the message that was wrritten to the buffer we definted above
		gotMsg := byteBuf.String()
		if gotMsg != tc.output {
			t.Errorf("expected stdout message to be: %v, got: %v\n", tc.output, gotMsg)
		}

		// call `Reset()` so that the buffer is emptied before executing the next test case
		byteBuf.Reset()
	}
}
