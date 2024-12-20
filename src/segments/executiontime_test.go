package segments

import (
	"math"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/properties"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime/mock"

	"github.com/stretchr/testify/assert"
)

func TestExecutionTimeWriterDefaultThresholdEnabled(t *testing.T) {
	env := new(mock.Environment)
	env.On("ExecutionTime").Return(1337)

	executionTime := &Executiontime{}
	executionTime.Init(properties.Map{}, env)

	assert.True(t, executionTime.Enabled())
}

func TestExecutionTimeWriterDefaultThresholdDisabled(t *testing.T) {
	env := new(mock.Environment)
	env.On("ExecutionTime").Return(1)

	executionTime := &Executiontime{}
	executionTime.Init(properties.Map{}, env)

	assert.False(t, executionTime.Enabled())
}

func TestExecutionTimeWriterCustomThresholdEnabled(t *testing.T) {
	env := new(mock.Environment)
	env.On("ExecutionTime").Return(99)
	props := properties.Map{
		ThresholdProperty: float64(10),
	}

	executionTime := &Executiontime{}
	executionTime.Init(props, env)

	assert.True(t, executionTime.Enabled())
}

func TestExecutionTimeWriterCustomThresholdDisabled(t *testing.T) {
	env := new(mock.Environment)
	env.On("ExecutionTime").Return(99)
	props := properties.Map{
		ThresholdProperty: float64(100),
	}

	executionTime := &Executiontime{}
	executionTime.Init(props, env)

	assert.False(t, executionTime.Enabled())
}

func TestExecutionTimeWriterDuration(t *testing.T) {
	input := 1337
	expected := "1.337s"
	env := new(mock.Environment)
	env.On("ExecutionTime").Return(input)

	executionTime := &Executiontime{}
	executionTime.Init(properties.Map{}, env)

	executionTime.Enabled()
	assert.Equal(t, expected, executionTime.FormattedMs)
}

func TestExecutionTimeWriterDuration2(t *testing.T) {
	input := 13371337
	expected := "3h 42m 51.337s"
	env := new(mock.Environment)
	env.On("ExecutionTime").Return(input)

	executionTime := &Executiontime{}
	executionTime.Init(properties.Map{}, env)

	executionTime.Enabled()
	assert.Equal(t, expected, executionTime.FormattedMs)
}

func TestExecutionTimeFormatDurationAustin(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "1ms"},
		{Input: "0.1s", Expected: "100ms"},
		{Input: "1s", Expected: "1s"},
		{Input: "2.1s", Expected: "2.1s"},
		{Input: "1m", Expected: "1m 0s"},
		{Input: "3m2.1s", Expected: "3m 2.1s"},
		{Input: "1h", Expected: "1h 0m 0s"},
		{Input: "4h3m2.1s", Expected: "4h 3m 2.1s"},
		{Input: "124h3m2.1s", Expected: "5d 4h 3m 2.1s"},
		{Input: "124h3m2.0s", Expected: "5d 4h 3m 2s"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationAustin()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatDurationRoundrock(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "1ms"},
		{Input: "0.1s", Expected: "100ms"},
		{Input: "1s", Expected: "1s 0ms"},
		{Input: "2.1s", Expected: "2s 100ms"},
		{Input: "1m", Expected: "1m 0s 0ms"},
		{Input: "3m2.1s", Expected: "3m 2s 100ms"},
		{Input: "1h", Expected: "1h 0m 0s 0ms"},
		{Input: "4h3m2.1s", Expected: "4h 3m 2s 100ms"},
		{Input: "124h3m2.1s", Expected: "5d 4h 3m 2s 100ms"},
		{Input: "124h3m2.0s", Expected: "5d 4h 3m 2s 0ms"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationRoundrock()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatDallas(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "0.001"},
		{Input: "0.1s", Expected: "0.1"},
		{Input: "1s", Expected: "1"},
		{Input: "2.1s", Expected: "2.1"},
		{Input: "1m", Expected: "1:0"},
		{Input: "3m2.1s", Expected: "3:2.1"},
		{Input: "1h", Expected: "1:0:0"},
		{Input: "4h3m2.1s", Expected: "4:3:2.1"},
		{Input: "124h3m2.1s", Expected: "5:4:3:2.1"},
		{Input: "124h3m2.0s", Expected: "5:4:3:2"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationDallas()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatGalveston(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "00:00:00"},
		{Input: "0.1s", Expected: "00:00:00"},
		{Input: "1s", Expected: "00:00:01"},
		{Input: "2.1s", Expected: "00:00:02"},
		{Input: "1m", Expected: "00:01:00"},
		{Input: "3m2.1s", Expected: "00:03:02"},
		{Input: "1h", Expected: "01:00:00"},
		{Input: "4h3m2.1s", Expected: "04:03:02"},
		{Input: "124h3m2.1s", Expected: "124:03:02"},
		{Input: "124h3m2.0s", Expected: "124:03:02"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationGalveston()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatGalvestonMs(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "00:00:00:001"},
		{Input: "0.1s", Expected: "00:00:00:100"},
		{Input: "1s", Expected: "00:00:01:000"},
		{Input: "2.1s", Expected: "00:00:02:100"},
		{Input: "1m", Expected: "00:01:00:000"},
		{Input: "3m2.1s", Expected: "00:03:02:100"},
		{Input: "1h", Expected: "01:00:00:000"},
		{Input: "4h3m2.1s", Expected: "04:03:02:100"},
		{Input: "124h3m2.1s", Expected: "124:03:02:100"},
		{Input: "124h3m2.0s", Expected: "124:03:02:000"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationGalvestonMs()
		assert.Equal(t, tc.Expected, output, tc.Input)
	}
}

func TestExecutionTimeFormatHouston(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "00:00:00.001"},
		{Input: "0.1s", Expected: "00:00:00.1"},
		{Input: "1s", Expected: "00:00:01.0"},
		{Input: "2.1s", Expected: "00:00:02.1"},
		{Input: "1m", Expected: "00:01:00.0"},
		{Input: "3m2.1s", Expected: "00:03:02.1"},
		{Input: "1h", Expected: "01:00:00.0"},
		{Input: "4h3m2.1s", Expected: "04:03:02.1"},
		{Input: "124h3m2.1s", Expected: "124:03:02.1"},
		{Input: "124h3m2.0s", Expected: "124:03:02.0"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationHouston()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatAmarillo(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "0.001s"},
		{Input: "0.1s", Expected: "0.1s"},
		{Input: "1s", Expected: "1s"},
		{Input: "2.1s", Expected: "2.1s"},
		{Input: "1m", Expected: "60s"},
		{Input: "3m2.1s", Expected: "182.1s"},
		{Input: "1h", Expected: "3,600s"},
		{Input: "4h3m2.1s", Expected: "14,582.1s"},
		{Input: "124h3m2.1s", Expected: "446,582.1s"},
		{Input: "124h3m2.0s", Expected: "446,582s"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationAmarillo()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatDurationRound(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{Input: "0.001s", Expected: "1ms"},
		{Input: "0.1s", Expected: "100ms"},
		{Input: "1s", Expected: "1s"},
		{Input: "2.1s", Expected: "2s"},
		{Input: "1m", Expected: "1m"},
		{Input: "3m2.1s", Expected: "3m 2s"},
		{Input: "1h", Expected: "1h"},
		{Input: "4h3m2.1s", Expected: "4h 3m"},
		{Input: "124h3m2.1s", Expected: "5d 4h"},
		{Input: "124h3m2.0s", Expected: "5d 4h"},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationRound()
		assert.Equal(t, tc.Expected, output)
	}
}

func TestExecutionTimeFormatDurationLucky7(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{
			Input:    "0.001s",
			Expected: "    1ms",
		},
		{
			Input:    "0.1s",
			Expected: "  100ms",
		},
		{
			Input:    "1s",
			Expected: " 1.00s ",
		},
		{
			Input:    "2.1s",
			Expected: " 2.10s ",
		},
		{
			Input:    "1m",
			Expected: " 1m  0s",
		},
		{
			Input:    "3m2.1s",
			Expected: " 3m  2s",
		},
		{
			Input:    "1h",
			Expected: " 1h  0m",
		},
		{
			Input:    "4h3m2.1s",
			Expected: " 4h  3m",
		},
		{
			Input:    "124h3m2.1s",
			Expected: " 5d  4h",
		},
		{
			Input:    "124h3m2.0s",
			Expected: " 5d  4h",
		},
	}

	for _, tc := range cases {
		duration, _ := time.ParseDuration(tc.Input)
		executionTime := &Executiontime{}
		executionTime.Ms = duration.Milliseconds()
		output := executionTime.formatDurationLucky7()
		assert.Equal(t, tc.Expected, output)
	}

	// Extra fuzz test
	var timestamp int64 = 1
	var ms1000days int64 = 1000 * 24 * 60 * 60 * 1000

	// log(ms1000days, 1.5) is approx 62.1
	for timestamp < ms1000days {
		timestamp = int64(math.Ceil(float64(timestamp) * 1.5))

		executionTime := (&Executiontime{
			Ms: timestamp,
		}).formatDurationLucky7()

		// Lucky 7!!
		assert.Equal(t, len(executionTime), 7)
	}
}
