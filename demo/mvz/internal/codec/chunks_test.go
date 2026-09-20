package codec

import (
	"bytes"
	"errors"
	"sync/atomic"
	"testing"
)

func TestChunkPipelineBounds(t *testing.T) {
	for _, test := range []struct {
		name string
		size int
		want int
	}{
		{name: "count", size: 1, want: 4},
		{name: "bytes", size: maxPendingChunkBytes/2 + 1, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkChunkPipelineBounds(t, test.size, test.want)
		})
	}
}

func checkChunkPipelineBounds(t *testing.T, size, want int) {
	t.Helper()
	var out bytes.Buffer
	pipeline := newChunkPipeline(&out, 2)
	for i := byte(0); i < 12; i++ {
		if err := pipeline.submit(size, func() ([]byte, error) {
			return []byte{i}, nil
		}); err != nil {
			t.Fatal(err)
		}
		if len(pipeline.pending) > want || pipeline.pendingBytes > maxPendingChunkBytes {
			t.Fatal("pending work exceeded its bound")
		}
		if out.Len() != max(0, int(i)+1-want) {
			t.Fatal("submit did not drain the bounded queue")
		}
	}
	if err := pipeline.finish(); err != nil {
		t.Fatal(err)
	}
	if len(pipeline.pending) != 0 || pipeline.pendingBytes != 0 {
		t.Fatal("finish retained pending chunks")
	}
	if !bytes.Equal(out.Bytes(), []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}) {
		t.Fatal("bounded pipeline changed output order")
	}
}

func TestChunkPipelineErrorOrderAndCleanup(t *testing.T) {
	var out bytes.Buffer
	var completed atomic.Int32
	pipeline := newChunkPipeline(&out, 2)
	firstErr := errors.New("first chunk")
	secondErr := errors.New("second chunk")
	secondDone := make(chan struct{})
	jobs := []func() ([]byte, error){
		func() ([]byte, error) {
			<-secondDone
			return nil, firstErr
		},
		func() ([]byte, error) {
			close(secondDone)
			return nil, secondErr
		},
		func() ([]byte, error) { return []byte("later output"), nil },
	}
	for _, job := range jobs {
		if err := pipeline.submit(1, func() ([]byte, error) {
			defer completed.Add(1)
			return job()
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := pipeline.finish(); err != firstErr {
		t.Fatalf("error %v, want first source-order error", err)
	}
	if out.Len() != 0 ||
		completed.Load() != int32(len(jobs)) ||
		len(pipeline.pending) != 0 {
		t.Fatal("failed pipeline wrote later output or left unfinished jobs")
	}
	if err := pipeline.submit(1, nil); err != firstErr {
		t.Fatalf("submission after error: %v", err)
	}
}
