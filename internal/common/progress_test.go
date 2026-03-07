package common

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConsoleOutput_ReturnsNonNil(t *testing.T) {
	out := NewConsoleOutput(&bytes.Buffer{}, false, false)
	require.NotNil(t, out)
}

func TestNewConsoleOutput_NilWriterDefaultsToStdout(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	out := NewConsoleOutput(nil, false, false)
	require.NotNil(t, out)
}

func TestNewConsoleOutput_NoColorFromParam(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	out := NewConsoleOutput(&bytes.Buffer{}, true, false)
	assert.True(t, out.noColor)
}

func TestNewConsoleOutput_NoColorFromEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	out := NewConsoleOutput(&bytes.Buffer{}, false, false)
	assert.True(t, out.noColor)
}

func TestNewConsoleOutput_QuietMode(t *testing.T) {
	out := NewConsoleOutput(&bytes.Buffer{}, true, true)
	assert.True(t, out.quiet)
}

func TestConsoleOutput_Info_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Info("hello %s", "world")

	output := buf.String()
	assert.Contains(t, output, "hello world")
}

func TestConsoleOutput_Info_HasInfoIndicator(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Info("test info")

	output := buf.String()
	assert.Contains(t, output, "test info")
}

func TestConsoleOutput_Info_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	out.Info("should not appear")

	assert.Empty(t, buf.String())
}

func TestConsoleOutput_Info_WithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Info("count: %d, name: %s", 42, "test")

	output := buf.String()
	assert.Contains(t, output, "count: 42, name: test")
}

func TestConsoleOutput_Success_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Success("all good")

	output := buf.String()
	assert.Contains(t, output, "all good")
}

func TestConsoleOutput_Success_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	out.Success("should not appear")

	assert.Empty(t, buf.String())
}

func TestConsoleOutput_Success_WithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Success("done: %d items", 5)

	assert.Contains(t, buf.String(), "done: 5 items")
}

func TestConsoleOutput_Warning_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Warning("careful here")

	assert.Contains(t, buf.String(), "careful here")
}

func TestConsoleOutput_Warning_NotSuppressedByQuiet(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	out.Warning("warning still shows")

	assert.Contains(t, buf.String(), "warning still shows")
}

func TestConsoleOutput_Warning_WithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Warning("timeout in %ds", 30)

	assert.Contains(t, buf.String(), "timeout in 30s")
}

func TestConsoleOutput_Error_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Error("something broke")

	assert.Contains(t, buf.String(), "something broke")
}

func TestConsoleOutput_Error_NotSuppressedByQuiet(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	out.Error("error still shows")

	assert.Contains(t, buf.String(), "error still shows")
}

func TestConsoleOutput_Error_WithFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Error("failed after %d retries: %s", 3, "timeout")

	assert.Contains(t, buf.String(), "failed after 3 retries: timeout")
}

func TestConsoleOutput_Step_WritesStepPrefix(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Step(2, 5, "Building contracts")

	output := buf.String()
	assert.Contains(t, output, "2/5")
	assert.Contains(t, output, "Building contracts")
}

func TestConsoleOutput_Step_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	out.Step(1, 3, "should not appear")

	assert.Empty(t, buf.String())
}

func TestConsoleOutput_NoColor_ProducesPlainText(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.Info("plain text")

	output := buf.String()
	assert.Contains(t, output, "plain text")
	assert.NotContains(t, output, "\033[")
}

func TestConsoleOutput_WithColor_ProducesANSI(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	out.Info("styled text")

	output := buf.String()
	assert.Contains(t, output, "\033[")
	assert.Contains(t, output, "styled text")
}

func TestConsoleOutput_Table_PrintsHeaders(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	headers := []string{"NAME", "STATUS"}
	rows := [][]string{
		{"chain1", "running"},
		{"chain2", "stopped"},
	}

	out.Table(headers, rows)

	output := buf.String()
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "STATUS")
}

func TestConsoleOutput_Table_PrintsRows(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	headers := []string{"NAME", "STATUS"}
	rows := [][]string{
		{"chain1", "running"},
		{"chain2", "stopped"},
	}

	out.Table(headers, rows)

	output := buf.String()
	assert.Contains(t, output, "chain1")
	assert.Contains(t, output, "running")
	assert.Contains(t, output, "chain2")
	assert.Contains(t, output, "stopped")
}

func TestConsoleOutput_Table_HasBorder(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	headers := []string{"A", "BB"}
	rows := [][]string{{"x", "yy"}}

	out.Table(headers, rows)

	output := buf.String()
	assert.True(t, strings.ContainsAny(output, "─┼├┤╭╮╰╯│┬┴+-|"), "expected border characters")
}

func TestConsoleOutput_Table_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	headers := []string{"NAME"}
	out.Table(headers, nil)

	output := buf.String()
	assert.Contains(t, output, "NAME")
}

func TestConsoleOutput_Table_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	headers := []string{"COL"}
	rows := [][]string{{"val"}}

	out.Table(headers, rows)

	output := buf.String()
	assert.Contains(t, output, "\033[")
}

func TestConsoleOutput_JSON_OutputsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	data := map[string]string{"key": "value"}
	out.JSON(data)

	output := strings.TrimSpace(buf.String())
	var parsed map[string]string
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "value", parsed["key"])
}

func TestConsoleOutput_JSON_PrettyPrinted(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	data := map[string]int{"a": 1}
	out.JSON(data)

	output := buf.String()
	assert.Contains(t, output, "\n")
	assert.Contains(t, output, "  ")
}

func TestConsoleOutput_JSON_InvalidValueFallsBack(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.JSON(make(chan int))

	output := buf.String()
	assert.Contains(t, output, "Failed to marshal JSON")
}

func TestConsoleOutput_Progress_ReturnsProgressBar(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	pb := out.Progress(10)
	require.NotNil(t, pb)
}

func TestConsoleProgressBar_Increment(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	pb := out.Progress(5)
	pb.Increment()
	pb.Increment()

	output := buf.String()
	assert.Contains(t, output, "2/5")
}

func TestConsoleProgressBar_SetCurrent(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	pb := out.Progress(10)
	pb.SetCurrent(7)

	output := buf.String()
	assert.Contains(t, output, "7/10")
}

func TestConsoleProgressBar_Done(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	pb := out.Progress(5)
	pb.Done()

	output := buf.String()
	assert.Contains(t, output, "5/5")
	assert.Contains(t, output, "100%")
}

func TestConsoleProgressBar_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	pb := out.Progress(0)
	pb.Increment()

	output := buf.String()
	assert.Contains(t, output, "0%")
}

func TestConsoleProgressBar_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	pb := out.Progress(10)
	pb.Increment()
	pb.SetCurrent(5)
	pb.Done()

	output := buf.String()
	assert.NotContains(t, output, "100%")
}

func TestConsoleOutput_Spinner_ReturnsSpinner(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("loading...")
	require.NotNil(t, s)
	s.Stop()
}

func TestConsoleSpinner_Stop(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("working")
	time.Sleep(20 * time.Millisecond)
	s.Stop()

	output := buf.String()
	assert.Contains(t, output, "\r\033[K")
}

func TestConsoleSpinner_StopIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("working")
	time.Sleep(20 * time.Millisecond)

	require.NotPanics(t, func() {
		s.Stop()
		s.Stop()
		s.Stop()
	})
}

func TestConsoleSpinner_StopWithSuccess(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("deploying")
	time.Sleep(20 * time.Millisecond)
	s.StopWithSuccess("deployed")

	output := buf.String()
	assert.Contains(t, output, "deployed")
}

func TestConsoleSpinner_StopWithError(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("deploying")
	time.Sleep(20 * time.Millisecond)
	s.StopWithError("deployment failed")

	output := buf.String()
	assert.Contains(t, output, "deployment failed")
}

func TestConsoleSpinner_Update(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("initial message")
	time.Sleep(20 * time.Millisecond)
	s.Update("new message")
	time.Sleep(150 * time.Millisecond) // Wait for a tick to render the new message.
	s.Stop()

	output := buf.String()
	assert.Contains(t, output, "new message")
}

func TestConsoleSpinner_QuietMode_NoGoroutine(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, true)

	s := out.Spinner("should be quiet")
	time.Sleep(150 * time.Millisecond) // Wait enough for a potential ticker tick.
	s.Stop()

	output := buf.String()
	for _, frame := range []string{"⠋", "⠙", "⠹", "⠸"} {
		assert.NotContains(t, output, frame)
	}
}

func TestConsoleSpinner_StopWithSuccessThenStopIsNoop(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	s := out.Spinner("task")
	time.Sleep(20 * time.Millisecond)
	s.StopWithSuccess("done")
	s.Stop()
	s.StopWithError("should not appear")

	output := buf.String()
	assert.Contains(t, output, "done")
	assert.NotContains(t, output, "should not appear")
}

func TestConsoleSpinner_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	s := out.Spinner("color spinner")
	time.Sleep(150 * time.Millisecond) // Allow at least one tick.
	s.Stop()

	output := buf.String()
	assert.Contains(t, output, "\033[")
}

func TestNewJSONOutput_ReturnsNonNil(t *testing.T) {
	out := NewJSONOutput(&bytes.Buffer{})
	require.NotNil(t, out)
}

func TestNewJSONOutput_NilWriterDefaultsToStdout(t *testing.T) {
	out := NewJSONOutput(nil)
	require.NotNil(t, out)
}

func TestJSONOutput_Info_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.Info("hello %s", "world")

	output := strings.TrimSpace(buf.String())
	var msg jsonMessage
	err := json.Unmarshal([]byte(output), &msg)
	require.NoError(t, err)
	assert.Equal(t, "info", msg.Level)
	assert.Equal(t, "hello world", msg.Message)
}

func TestJSONOutput_Success_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.Success("done: %d", 42)

	output := strings.TrimSpace(buf.String())
	var msg jsonMessage
	err := json.Unmarshal([]byte(output), &msg)
	require.NoError(t, err)
	assert.Equal(t, "success", msg.Level)
	assert.Equal(t, "done: 42", msg.Message)
}

func TestJSONOutput_Warning_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.Warning("be careful")

	output := strings.TrimSpace(buf.String())
	var msg jsonMessage
	err := json.Unmarshal([]byte(output), &msg)
	require.NoError(t, err)
	assert.Equal(t, "warning", msg.Level)
	assert.Equal(t, "be careful", msg.Message)
}

func TestJSONOutput_Error_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.Error("something broke: %s", "oom")

	output := strings.TrimSpace(buf.String())
	var msg jsonMessage
	err := json.Unmarshal([]byte(output), &msg)
	require.NoError(t, err)
	assert.Equal(t, "error", msg.Level)
	assert.Equal(t, "something broke: oom", msg.Message)
}

func TestJSONOutput_Step_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.Step(2, 5, "Building")

	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "step", parsed["level"])
	assert.Equal(t, float64(2), parsed["current"])
	assert.Equal(t, float64(5), parsed["total"])
	assert.Equal(t, "Building", parsed["message"])
}

func TestJSONOutput_Table_WritesJSONArray(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	headers := []string{"name", "status"}
	rows := [][]string{
		{"chain1", "running"},
		{"chain2", "stopped"},
	}

	out.Table(headers, rows)

	output := strings.TrimSpace(buf.String())
	var parsed []map[string]string
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	require.Len(t, parsed, 2)
	assert.Equal(t, "chain1", parsed[0]["name"])
	assert.Equal(t, "running", parsed[0]["status"])
	assert.Equal(t, "chain2", parsed[1]["name"])
	assert.Equal(t, "stopped", parsed[1]["status"])
}

func TestJSONOutput_Table_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	headers := []string{"name"}
	out.Table(headers, nil)

	output := strings.TrimSpace(buf.String())
	var parsed []map[string]string
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Empty(t, parsed)
}

func TestJSONOutput_Table_RowShorterThanHeaders(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	headers := []string{"a", "b", "c"}
	rows := [][]string{
		{"only_a"},
	}

	out.Table(headers, rows)

	output := strings.TrimSpace(buf.String())
	var parsed []map[string]string
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	assert.Equal(t, "only_a", parsed[0]["a"])
	_, hasB := parsed[0]["b"]
	assert.False(t, hasB)
}

func TestJSONOutput_Progress_ReturnsNoopProgressBar(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	pb := out.Progress(10)
	require.NotNil(t, pb)

	require.NotPanics(t, func() {
		pb.Increment()
		pb.SetCurrent(5)
		pb.Done()
	})
}

func TestJSONOutput_Spinner_ReturnsNoopSpinner(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	s := out.Spinner("loading")
	require.NotNil(t, s)

	require.NotPanics(t, func() {
		s.Update("new msg")
		s.Stop()
		s.StopWithSuccess("done")
		s.StopWithError("failed")
	})
}

func TestJSONOutput_JSON_OutputsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	data := map[string]int{"count": 42}
	out.JSON(data)

	output := strings.TrimSpace(buf.String())
	var parsed map[string]int
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Equal(t, 42, parsed["count"])
}

func TestJSONOutput_JSON_PrettyPrinted(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	data := map[string]string{"key": "value"}
	out.JSON(data)

	output := buf.String()
	assert.Contains(t, output, "\n")
	assert.Contains(t, output, "  ")
}

func TestJSONOutput_JSON_InvalidValueFallback(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.JSON(make(chan int))

	output := buf.String()
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "failed to marshal JSON")
}

func TestConsoleOutput_ImplementsOutputInterface(t *testing.T) {
	var _ Output = (*ConsoleOutput)(nil)
}

func TestJSONOutput_ImplementsOutputInterface(t *testing.T) {
	var _ Output = (*JSONOutput)(nil)
}

func TestConsoleProgressBar_ImplementsProgressBarInterface(t *testing.T) {
	var _ ProgressBar = (*consoleProgressBar)(nil)
}

func TestNoopProgressBar_ImplementsProgressBarInterface(t *testing.T) {
	var _ ProgressBar = (*noopProgressBar)(nil)
}

func TestConsoleSpinner_ImplementsSpinnerInterface(t *testing.T) {
	var _ Spinner = (*consoleSpinner)(nil)
}

func TestNoopSpinner_ImplementsSpinnerInterface(t *testing.T) {
	var _ Spinner = (*noopSpinner)(nil)
}

func TestNoopProgressBar_AllMethodsAreNoOps(t *testing.T) {
	pb := &noopProgressBar{}
	pb.Increment()
	pb.SetCurrent(99)
	pb.Done()
}

func TestNoopSpinner_AllMethodsAreNoOps(t *testing.T) {
	s := &noopSpinner{}
	s.Update("msg")
	s.Stop()
	s.StopWithSuccess("ok")
	s.StopWithError("err")
}

func TestConsoleOutput_Info_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	out.Info("colored info")

	output := buf.String()
	assert.Contains(t, output, "\033[")
	assert.Contains(t, output, "colored info")
}

func TestConsoleOutput_Success_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	out.Success("colored success")

	output := buf.String()
	assert.Contains(t, output, "\033[")
}

func TestConsoleOutput_Warning_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	out.Warning("colored warning")

	output := buf.String()
	assert.Contains(t, output, "\033[")
}

func TestConsoleOutput_Error_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	out.Error("colored error")

	output := buf.String()
	assert.Contains(t, output, "\033[")
}

func TestConsoleOutput_Step_WithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")

	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, false, false)

	out.Step(1, 3, "step msg")

	output := buf.String()
	assert.Contains(t, output, "\033[")
	assert.Contains(t, output, "1/3")
}

func TestConsoleProgressBar_IncrementBeyondTotal(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	pb := out.Progress(2)
	pb.Increment()
	pb.Increment()
	pb.Increment() // Beyond total -- should not panic.

	output := buf.String()
	assert.Contains(t, output, "3/2")
}

func TestConsoleOutput_JSON_NilValue(t *testing.T) {
	var buf bytes.Buffer
	out := NewConsoleOutput(&buf, true, false)

	out.JSON(nil)

	output := strings.TrimSpace(buf.String())
	assert.Equal(t, "null", output)
}

func TestJSONOutput_JSON_NilValue(t *testing.T) {
	var buf bytes.Buffer
	out := NewJSONOutput(&buf)

	out.JSON(nil)

	output := strings.TrimSpace(buf.String())
	assert.Equal(t, "null", output)
}
