package codec

import (
	"fmt"
	"io"
	"sync"
)

const maxPendingChunkBytes = 2 * maxChunkSize

type chunkResult struct {
	data []byte
	err  error
}

type chunkJob struct {
	work   func() ([]byte, error)
	result chan chunkResult
}

type pendingChunk struct {
	result chan chunkResult
	size   int
}

// chunkPipeline writes results in submission order. Its limits include chunks
// waiting for a worker, running, or waiting to be written, so a slow first job
// cannot create an unbounded reorder buffer. Only the caller accesses pending or dst.
// Callers charge original MVD bytes when encoding, or encoded plus advertised
// decoded bytes when decoding. This is a queue budget, not a Go heap limit.
type chunkPipeline struct {
	dst          io.Writer
	jobs         chan chunkJob
	wait         sync.WaitGroup
	pending      []pendingChunk
	pendingBytes int
	pendingLimit int
	err          error
}

func newChunkPipeline(dst io.Writer, workers int) *chunkPipeline {
	workers = max(1, workers)
	pipeline := &chunkPipeline{
		dst:          dst,
		jobs:         make(chan chunkJob, workers),
		pendingLimit: 2 * workers,
	}
	for range workers {
		pipeline.wait.Add(1)
		go pipeline.run()
	}
	return pipeline
}

func (pipeline *chunkPipeline) run() {
	defer pipeline.wait.Done()
	for job := range pipeline.jobs {
		data, err := job.work()
		job.result <- chunkResult{data: data, err: err}
	}
}

func (pipeline *chunkPipeline) submit(size int, work func() ([]byte, error)) error {
	if pipeline.err != nil {
		return pipeline.err
	}
	if size < 0 || size > maxPendingChunkBytes {
		return fmt.Errorf("invalid MVZ pending chunk size %d", size)
	}
	for len(pipeline.pending) >= pipeline.pendingLimit ||
		pipeline.pendingBytes > maxPendingChunkBytes-size {
		pipeline.writeNext()
		if pipeline.err != nil {
			return pipeline.err
		}
	}
	result := make(chan chunkResult, 1)
	pipeline.jobs <- chunkJob{work: work, result: result}
	pipeline.pending = append(pipeline.pending, pendingChunk{result: result, size: size})
	pipeline.pendingBytes += size
	return nil
}

func (pipeline *chunkPipeline) writeNext() {
	chunk := pipeline.pending[0]
	result := <-chunk.result
	pipeline.pending[0] = pendingChunk{}
	pipeline.pending = pipeline.pending[1:]
	pipeline.pendingBytes -= chunk.size
	if pipeline.err != nil {
		return
	}
	if result.err != nil {
		pipeline.err = result.err
		return
	}
	n, err := pipeline.dst.Write(result.data)
	if err == nil && n != len(result.data) {
		err = io.ErrShortWrite
	}
	pipeline.err = err
}

// finish must be called once, even when the producer fails. It waits for every
// submitted job, returns the first error in source order, and stops writing
// after that error. No worker can outlive the encode/decode call.
func (pipeline *chunkPipeline) finish() error {
	close(pipeline.jobs)
	for len(pipeline.pending) != 0 {
		pipeline.writeNext()
	}
	pipeline.wait.Wait()
	return pipeline.err
}
