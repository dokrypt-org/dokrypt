package container

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageManager(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewImageManager(mock)
	require.NotNil(t, mgr)
	assert.Equal(t, mock, mgr.runtime)
}

func TestPullOptions_Defaults(t *testing.T) {
	opts := PullOptions{}
	assert.False(t, opts.Force)
	assert.Nil(t, opts.OnProgress)
}

func TestPullOptions_WithValues(t *testing.T) {
	called := false
	opts := PullOptions{
		Force: true,
		OnProgress: func(status string, pct int) {
			called = true
		},
	}
	assert.True(t, opts.Force)
	opts.OnProgress("test", 50)
	assert.True(t, called)
}

func TestImageManager_Exists_ByTag(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "sha256:aaa", Tags: []string{"alpine:3.18", "alpine:latest"}},
			{ID: "sha256:bbb", Tags: []string{"nginx:latest"}},
		}, nil
	}

	mgr := NewImageManager(mock)

	exists, err := mgr.Exists(context.Background(), "alpine:3.18")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = mgr.Exists(context.Background(), "nginx:latest")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestImageManager_Exists_ByID(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "sha256:aaa", Tags: []string{"alpine:latest"}},
		}, nil
	}

	mgr := NewImageManager(mock)
	exists, err := mgr.Exists(context.Background(), "sha256:aaa")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestImageManager_Exists_NotFound(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "sha256:aaa", Tags: []string{"alpine:latest"}},
		}, nil
	}

	mgr := NewImageManager(mock)
	exists, err := mgr.Exists(context.Background(), "ubuntu:latest")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestImageManager_Exists_EmptyList(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}

	mgr := NewImageManager(mock)
	exists, err := mgr.Exists(context.Background(), "anything")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestImageManager_Exists_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return nil, errMock
	}

	mgr := NewImageManager(mock)
	exists, err := mgr.Exists(context.Background(), "alpine")
	require.Error(t, err)
	assert.False(t, exists)
}

func TestImageManager_Pull_ImageAlreadyExists(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "sha256:abc", Tags: []string{"alpine:latest"}},
		}, nil
	}

	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "alpine:latest", PullOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.calls["PullImage"])
}

func TestImageManager_Pull_ForceEvenIfExists(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "sha256:abc", Tags: []string{"alpine:latest"}},
		}, nil
	}

	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "alpine:latest", PullOptions{Force: true})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["PullImage"])
}

func TestImageManager_Pull_ImageNotExists(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}

	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "alpine:latest", PullOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["PullImage"])
}

func TestImageManager_Pull_ListFailsPullsAnyway(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return nil, errMock
	}

	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "alpine:latest", PullOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["PullImage"])
}

func TestImageManager_Pull_PullError(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}
	mock.pullImageFn = func(ctx context.Context, image string) error {
		return fmt.Errorf("network error")
	}

	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "alpine:latest", PullOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to pull image")
	assert.Contains(t, err.Error(), "network error")
}

func TestImageManager_Pull_ProgressCallback(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}

	var progressCalls []struct {
		status string
		pct    int
	}

	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "nginx:latest", PullOptions{
		OnProgress: func(status string, pct int) {
			progressCalls = append(progressCalls, struct {
				status string
				pct    int
			}{status, pct})
		},
	})
	require.NoError(t, err)
	require.Len(t, progressCalls, 2)
	assert.Equal(t, "pulling nginx:latest", progressCalls[0].status)
	assert.Equal(t, 0, progressCalls[0].pct)
	assert.Equal(t, "pulled nginx:latest", progressCalls[1].status)
	assert.Equal(t, 100, progressCalls[1].pct)
}

func TestImageManager_Pull_NoProgressCallbackOnExistingImage(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "sha256:abc", Tags: []string{"alpine:latest"}},
		}, nil
	}

	called := false
	mgr := NewImageManager(mock)
	err := mgr.Pull(context.Background(), "alpine:latest", PullOptions{
		OnProgress: func(status string, pct int) {
			called = true
		},
	})
	require.NoError(t, err)
	assert.False(t, called, "progress should not be called when image exists")
}

func TestImageManager_Build_Success(t *testing.T) {
	mock := newMockRuntime()
	mock.buildImageFn = func(ctx context.Context, contextPath string, opts BuildOptions) (string, error) {
		return "myimage:v1", nil
	}

	mgr := NewImageManager(mock)
	id, err := mgr.Build(context.Background(), "/path/to/context", BuildOptions{
		Tags: []string{"myimage:v1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "myimage:v1", id)
	assert.Equal(t, 1, mock.calls["BuildImage"])
}

func TestImageManager_Build_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.buildImageFn = func(ctx context.Context, contextPath string, opts BuildOptions) (string, error) {
		return "", fmt.Errorf("build failed")
	}

	mgr := NewImageManager(mock)
	id, err := mgr.Build(context.Background(), "/ctx", BuildOptions{})
	require.Error(t, err)
	assert.Empty(t, id)
}

func TestImageManager_List_NoFilter(t *testing.T) {
	now := time.Now()
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "1", Tags: []string{"alpine:latest"}, Size: 1000, Created: now},
			{ID: "2", Tags: []string{"nginx:latest"}, Size: 2000, Created: now},
		}, nil
	}

	mgr := NewImageManager(mock)
	images, err := mgr.List(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, images, 2)
}

func TestImageManager_List_WithFilter(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "1", Tags: []string{"alpine:3.18", "alpine:latest"}},
			{ID: "2", Tags: []string{"nginx:latest"}},
			{ID: "3", Tags: []string{"alpine-node:14"}},
		}, nil
	}

	mgr := NewImageManager(mock)
	images, err := mgr.List(context.Background(), "alpine")
	require.NoError(t, err)
	assert.Len(t, images, 2) // "alpine:3.18" and "alpine-node:14" both start with "alpine"
}

func TestImageManager_List_FilterNoMatch(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{
			{ID: "1", Tags: []string{"alpine:latest"}},
		}, nil
	}

	mgr := NewImageManager(mock)
	images, err := mgr.List(context.Background(), "ubuntu")
	require.NoError(t, err)
	assert.Len(t, images, 0)
}

func TestImageManager_List_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return nil, errMock
	}

	mgr := NewImageManager(mock)
	images, err := mgr.List(context.Background(), "")
	require.Error(t, err)
	assert.Nil(t, images)
}

func TestImageManager_List_EmptyResult(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}

	mgr := NewImageManager(mock)
	images, err := mgr.List(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, images)
}

func TestImageManager_Remove_Success(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewImageManager(mock)

	err := mgr.Remove(context.Background(), "alpine:latest", false)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["RemoveImage"])
}

func TestImageManager_Remove_Force(t *testing.T) {
	mock := newMockRuntime()
	var capturedForce bool
	mock.removeImageFn = func(ctx context.Context, image string, force bool) error {
		capturedForce = force
		return nil
	}

	mgr := NewImageManager(mock)
	err := mgr.Remove(context.Background(), "alpine:latest", true)
	require.NoError(t, err)
	assert.True(t, capturedForce)
}

func TestImageManager_Remove_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.removeImageFn = func(ctx context.Context, image string, force bool) error {
		return errMock
	}

	mgr := NewImageManager(mock)
	err := mgr.Remove(context.Background(), "alpine:latest", false)
	require.Error(t, err)
}

func TestImageManager_PullParallel_AllSucceed(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}

	mgr := NewImageManager(mock)
	err := mgr.PullParallel(context.Background(), []string{"alpine:latest", "nginx:latest"}, PullOptions{Force: true})
	require.NoError(t, err)
	assert.Equal(t, 2, mock.calls["PullImage"])
}

func TestImageManager_PullParallel_OneFailure(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}
	var callCount int32
	mock.pullImageFn = func(ctx context.Context, image string) error {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			return fmt.Errorf("pull failed for %s", image)
		}
		return nil
	}

	mgr := NewImageManager(mock)
	err := mgr.PullParallel(context.Background(), []string{"bad:image", "good:image"}, PullOptions{Force: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to pull image")
}

func TestImageManager_PullParallel_EmptyList(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewImageManager(mock)
	err := mgr.PullParallel(context.Background(), []string{}, PullOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.calls["PullImage"])
}

func TestImageManager_PullParallel_SingleImage(t *testing.T) {
	mock := newMockRuntime()
	mock.listImagesFn = func(ctx context.Context) ([]ImageInfo, error) {
		return []ImageInfo{}, nil
	}

	mgr := NewImageManager(mock)
	err := mgr.PullParallel(context.Background(), []string{"alpine:latest"}, PullOptions{Force: true})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["PullImage"])
}
