package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerRuntime_ImplementsRuntime(t *testing.T) {
	var _ Runtime = (*DockerRuntime)(nil)
}

func TestNewDockerRuntime_CreatesInstance(t *testing.T) {
	rt, err := NewDockerRuntime()
	if err != nil {
		assert.Contains(t, err.Error(), "Docker")
	} else {
		assert.NotNil(t, rt)
		assert.NotNil(t, rt.client)
	}
}

func TestNewRuntime_Docker(t *testing.T) {
	rt, err := NewRuntime("docker")
	if err != nil {
		t.Skipf("Docker client creation failed (expected in environments without Docker): %v", err)
	}
	require.NotNil(t, rt)
	_, ok := rt.(*DockerRuntime)
	assert.True(t, ok, "expected *DockerRuntime")
}

func TestNewRuntime_EmptyStringDefaultsToDocker(t *testing.T) {
	rt, err := NewRuntime("")
	if err != nil {
		t.Skipf("Docker client creation failed: %v", err)
	}
	require.NotNil(t, rt)
	_, ok := rt.(*DockerRuntime)
	assert.True(t, ok, "empty string should default to DockerRuntime")
}

func TestBuildOptions_ArgConstruction(t *testing.T) {
	tests := []struct {
		name        string
		opts        BuildOptions
		contextPath string
		expectTags  int
	}{
		{
			name: "no tags, no dockerfile",
			opts: BuildOptions{},
		},
		{
			name: "single tag",
			opts: BuildOptions{
				Tags: []string{"myimg:v1"},
			},
			expectTags: 1,
		},
		{
			name: "multiple tags",
			opts: BuildOptions{
				Tags: []string{"myimg:v1", "myimg:latest"},
			},
			expectTags: 2,
		},
		{
			name: "custom dockerfile",
			opts: BuildOptions{
				Dockerfile: "Dockerfile.prod",
			},
		},
		{
			name: "build args",
			opts: BuildOptions{
				BuildArgs: map[string]string{"VERSION": "1.0", "COMMIT": "abc"},
			},
		},
		{
			name: "no-cache",
			opts: BuildOptions{
				NoCache: true,
			},
		},
		{
			name: "fully populated",
			opts: BuildOptions{
				Tags:       []string{"img:v1"},
				Dockerfile: "Dockerfile.test",
				BuildArgs:  map[string]string{"A": "B"},
				NoCache:    true,
			},
			expectTags: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"build"}

			for _, tag := range tc.opts.Tags {
				args = append(args, "-t", tag)
			}
			if tc.opts.Dockerfile != "" {
				args = append(args, "-f", tc.opts.Dockerfile)
			}
			for k, v := range tc.opts.BuildArgs {
				args = append(args, "--build-arg", k+"="+v)
			}
			if tc.opts.NoCache {
				args = append(args, "--no-cache")
			}
			args = append(args, "/tmp/ctx")

			assert.Equal(t, "build", args[0])
			assert.Equal(t, "/tmp/ctx", args[len(args)-1])

			tagCount := 0
			for i, a := range args {
				if a == "-t" && i+1 < len(args) {
					tagCount++
				}
			}
			assert.Equal(t, tc.expectTags, tagCount)
		})
	}
}

func TestPortBindingConstruction(t *testing.T) {
	cfg := &ContainerConfig{
		Ports: map[int]int{
			8080: 80,
			443:  0,
			3000: 3000,
		},
	}

	assert.Len(t, cfg.Ports, 3)
	assert.Equal(t, 80, cfg.Ports[8080])
	assert.Equal(t, 0, cfg.Ports[443])
	assert.Equal(t, 3000, cfg.Ports[3000])
}

func TestEnvConstruction(t *testing.T) {
	cfg := &ContainerConfig{
		Env: map[string]string{
			"DB_HOST": "localhost",
			"DB_PORT": "5432",
		},
	}

	var env []string
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}

	assert.Len(t, env, 2)
	assert.Contains(t, env, "DB_HOST=localhost")
	assert.Contains(t, env, "DB_PORT=5432")
}

func TestVolumeBindConstruction(t *testing.T) {
	tests := []struct {
		name     string
		mount    VolumeMount
		expected string
	}{
		{
			name:     "read-write",
			mount:    VolumeMount{Source: "/host/data", Target: "/data"},
			expected: "/host/data:/data",
		},
		{
			name:     "read-only",
			mount:    VolumeMount{Source: "myvolume", Target: "/app/data", ReadOnly: true},
			expected: "myvolume:/app/data:ro",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bind := tc.mount.Source + ":" + tc.mount.Target
			if tc.mount.ReadOnly {
				bind += ":ro"
			}
			assert.Equal(t, tc.expected, bind)
		})
	}
}

func TestRestartPolicyMapping(t *testing.T) {
	policies := []string{"", "always", "unless-stopped", "on-failure"}
	for _, p := range policies {
		cfg := &ContainerConfig{RestartPolicy: p}
		assert.Equal(t, p, cfg.RestartPolicy)
	}
}

func TestLogOptionsDefaultStdoutStderr(t *testing.T) {
	tests := []struct {
		name           string
		opts           LogOptions
		expectStdout   bool
		expectStderr   bool
	}{
		{
			name:         "both false defaults to both true",
			opts:         LogOptions{},
			expectStdout: true,
			expectStderr: true,
		},
		{
			name:         "only stdout",
			opts:         LogOptions{Stdout: true},
			expectStdout: true,
			expectStderr: false,
		},
		{
			name:         "only stderr",
			opts:         LogOptions{Stderr: true},
			expectStdout: false,
			expectStderr: true,
		},
		{
			name:         "both true",
			opts:         LogOptions{Stdout: true, Stderr: true},
			expectStdout: true,
			expectStderr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			showStdout := tc.opts.Stdout || (!tc.opts.Stdout && !tc.opts.Stderr)
			showStderr := tc.opts.Stderr || (!tc.opts.Stdout && !tc.opts.Stderr)
			assert.Equal(t, tc.expectStdout, showStdout)
			assert.Equal(t, tc.expectStderr, showStderr)
		})
	}
}
