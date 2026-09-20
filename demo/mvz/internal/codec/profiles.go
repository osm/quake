package codec

import (
	"fmt"
	"sync"
)

type rangeProfileResult struct {
	rangeData []byte
	err       error
}

func tryProfiles(
	input *rangeProfileInput,
	resources *RangeResources,
	workers int,
) ([][]byte, error) {
	workers = max(1, min(workers, len(resources.profiles)))
	results := make([]rangeProfileResult, len(resources.profiles))
	encoder := rangeProfileEncoder{
		input:     input,
		resources: resources,
		results:   results,
	}
	encoder.encode(workers)

	rangeStreams := make([][]byte, len(results))
	for i, result := range results {
		if result.err != nil {
			return nil, fmt.Errorf(
				"encode MVZ frequency profile %d: %w",
				resources.profiles[i].id,
				result.err,
			)
		}
		rangeStreams[i] = result.rangeData
	}
	return rangeStreams, nil
}

type rangeProfileEncoder struct {
	input     *rangeProfileInput
	resources *RangeResources
	results   []rangeProfileResult
	jobs      chan int
}

func (encoder *rangeProfileEncoder) encode(workers int) {
	if workers == 1 {
		for index := range encoder.resources.profiles {
			encoder.tryProfile(index)
		}
		return
	}
	encoder.jobs = make(chan int)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go encoder.run(&wait)
	}
	for index := range encoder.resources.profiles {
		encoder.jobs <- index
	}
	close(encoder.jobs)
	wait.Wait()
}

func (encoder *rangeProfileEncoder) run(wait *sync.WaitGroup) {
	defer wait.Done()
	for index := range encoder.jobs {
		encoder.tryProfile(index)
	}
}

func (encoder *rangeProfileEncoder) tryProfile(index int) {
	profile := encoder.resources.profiles[index]
	result := &encoder.results[index]
	result.rangeData, result.err = encodeRangeStream(encoder.input, profile.models)
}
