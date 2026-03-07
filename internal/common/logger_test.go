package common

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevel_Debug(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, ParseLogLevel("debug"))
}

func TestParseLogLevel_DebugUppercase(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, ParseLogLevel("DEBUG"))
}

func TestParseLogLevel_DebugMixedCase(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, ParseLogLevel("Debug"))
}

func TestParseLogLevel_Info(t *testing.T) {
	assert.Equal(t, slog.LevelInfo, ParseLogLevel("info"))
}

func TestParseLogLevel_InfoUppercase(t *testing.T) {
	assert.Equal(t, slog.LevelInfo, ParseLogLevel("INFO"))
}

func TestParseLogLevel_Warn(t *testing.T) {
	assert.Equal(t, slog.LevelWarn, ParseLogLevel("warn"))
}

func TestParseLogLevel_Warning(t *testing.T) {
	assert.Equal(t, slog.LevelWarn, ParseLogLevel("warning"))
}

func TestParseLogLevel_WarningUppercase(t *testing.T) {
	assert.Equal(t, slog.LevelWarn, ParseLogLevel("WARNING"))
}

func TestParseLogLevel_Error(t *testing.T) {
	assert.Equal(t, slog.LevelError, ParseLogLevel("error"))
}

func TestParseLogLevel_ErrorUppercase(t *testing.T) {
	assert.Equal(t, slog.LevelError, ParseLogLevel("ERROR"))
}

func TestParseLogLevel_UnknownDefaultsToInfo(t *testing.T) {
	assert.Equal(t, slog.LevelInfo, ParseLogLevel("unknown"))
}

func TestParseLogLevel_EmptyDefaultsToInfo(t *testing.T) {
	assert.Equal(t, slog.LevelInfo, ParseLogLevel(""))
}

func TestParseLogLevel_GarbageDefaultsToInfo(t *testing.T) {
	assert.Equal(t, slog.LevelInfo, ParseLogLevel("foobar123"))
}

func TestLogLevelConstants(t *testing.T) {
	assert.Equal(t, LogLevel("debug"), LogLevelDebug)
	assert.Equal(t, LogLevel("info"), LogLevelInfo)
	assert.Equal(t, LogLevel("warn"), LogLevelWarn)
	assert.Equal(t, LogLevel("error"), LogLevelError)
}

func TestSetupLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("info", false, &buf)

	slog.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
}

func TestSetupLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("info", true, &buf)

	slog.Info("json test", "num", 42)

	output := buf.String()
	assert.Contains(t, output, "json test")
	assert.Contains(t, output, `"num"`)
	assert.Contains(t, output, "42")
}

func TestSetupLogger_DebugLevelAllowsDebugMessages(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("debug", false, &buf)

	slog.Debug("debug visible")

	output := buf.String()
	assert.Contains(t, output, "debug visible")
}

func TestSetupLogger_InfoLevelFiltersDebugMessages(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("info", false, &buf)

	slog.Debug("debug should not appear")

	output := buf.String()
	assert.NotContains(t, output, "debug should not appear")
}

func TestSetupLogger_ErrorLevelFiltersLowerMessages(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("error", false, &buf)

	slog.Info("info should not appear")
	slog.Warn("warn should not appear")

	output := buf.String()
	assert.NotContains(t, output, "info should not appear")
	assert.NotContains(t, output, "warn should not appear")
}

func TestSetupLogger_ErrorLevelShowsErrors(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("error", false, &buf)

	slog.Error("error visible")

	output := buf.String()
	assert.Contains(t, output, "error visible")
}

func TestSetupLogger_NilWriterDefaultsToStderr(t *testing.T) {
	require.NotPanics(t, func() {
		SetupLogger("info", false, nil)
	})
}

func TestSetupLogger_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("warn", false, &buf)

	slog.Info("info filtered")
	slog.Warn("warn visible")

	output := buf.String()
	assert.NotContains(t, output, "info filtered")
	assert.Contains(t, output, "warn visible")
}

func TestSetupLogger_JSONFormatWithDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	SetupLogger("debug", true, &buf)

	slog.Debug("json debug msg")

	output := buf.String()
	assert.Contains(t, output, "json debug msg")
	assert.Contains(t, output, `"level"`)
}

func TestIsCI_DokryptCITrue(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "true")
	t.Setenv("CI", "")
	assert.True(t, IsCI())
}

func TestIsCI_CITrue(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "true")
	assert.True(t, IsCI())
}

func TestIsCI_BothTrue(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "true")
	t.Setenv("CI", "true")
	assert.True(t, IsCI())
}

func TestIsCI_NeitherSet(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.False(t, IsCI())
}

func TestIsCI_DokryptCIFalse(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "false")
	t.Setenv("CI", "")
	assert.False(t, IsCI())
}

func TestIsCI_CIFalse(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "false")
	assert.False(t, IsCI())
}

func TestIsCI_CIYes(t *testing.T) {
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "yes")
	assert.False(t, IsCI())
}

func TestNoColor_NOCOLORSet(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.True(t, NoColor())
}

func TestNoColor_NOCOLORAnyValue(t *testing.T) {
	t.Setenv("NO_COLOR", "anything")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.True(t, NoColor())
}

func TestNoColor_DokryptNoColorTrue(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "true")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.True(t, NoColor())
}

func TestNoColor_InCI(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "true")
	t.Setenv("CI", "")
	assert.True(t, NoColor())
}

func TestNoColor_InCIEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "true")
	assert.True(t, NoColor())
}

func TestNoColor_NothingSet(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.False(t, NoColor())
}

func TestNoColor_DokryptNoColorFalse(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "false")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.False(t, NoColor())
}

func TestNoColor_NOCOLOREmpty(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("DOKRYPT_NO_COLOR", "")
	t.Setenv("DOKRYPT_CI", "")
	t.Setenv("CI", "")
	assert.False(t, NoColor())
}
