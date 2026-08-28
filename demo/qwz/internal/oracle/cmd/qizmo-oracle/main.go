package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/osm/quake/demo/qwz/internal/oracle"
)

type configuration struct {
	qizmo       string
	compressDat string
	seed        string
	output      string
	workers     int
	limit       int
	family      string
	name        string
	artifacts   string
}

func main() {
	var cfg configuration
	flag.StringVar(&cfg.qizmo, "qizmo", "", "path to the original Qizmo v2.91 binary")
	flag.StringVar(&cfg.compressDat, "compress-dat", "", "path to Qizmo's original compress.dat")
	flag.StringVar(&cfg.seed, "seed", "demo/qwz/testdata/demo26.qwz", "seed QWZ fixture")
	flag.StringVar(&cfg.output, "output", "", "output JSON manifest")
	flag.IntVar(&cfg.workers, "workers", min(runtime.NumCPU(), 8), "parallel isolated Qizmo processes")
	flag.IntVar(&cfg.limit, "limit", 0, "generate only the first N selected scenarios (development checks)")
	flag.StringVar(&cfg.family, "family", "", "generate only one scenario family")
	flag.StringVar(&cfg.name, "name", "", "generate only one exact scenario name")
	flag.StringVar(&cfg.artifacts, "artifacts", "", "optional directory for generated QWD/QWZ artifacts")
	flag.Parse()
	if cfg.qizmo == "" || cfg.compressDat == "" || cfg.output == "" || cfg.workers < 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg configuration) error {
	seed, err := os.ReadFile(cfg.seed)
	if err != nil {
		return fmt.Errorf("read seed: %w", err)
	}
	scenarios, err := oracle.Build(seed)
	if err != nil {
		return err
	}
	scenarios = selectScenarios(scenarios, cfg.family, cfg.name)
	if cfg.limit > 0 && cfg.limit < len(scenarios) {
		scenarios = scenarios[:cfg.limit]
	}
	if len(scenarios) == 0 {
		return fmt.Errorf("no scenarios selected")
	}
	if cfg.artifacts != "" {
		if err := os.MkdirAll(cfg.artifacts, 0o755); err != nil {
			return fmt.Errorf("create artifacts directory: %w", err)
		}
	}
	compressDat, err := os.ReadFile(cfg.compressDat)
	if err != nil {
		return fmt.Errorf("read compress.dat: %w", err)
	}
	qizmo, err := filepath.Abs(cfg.qizmo)
	if err != nil {
		return err
	}

	results := make([]oracle.Result, len(scenarios))
	jobs := make(chan int, len(scenarios))
	for index := range scenarios {
		jobs <- index
	}
	close(jobs)
	errCh := make(chan error, 1)
	var completed atomic.Int64
	var wg sync.WaitGroup
	for range cfg.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workDir, err := prepareWorkDir(compressDat)
			if err != nil {
				reportError(errCh, err)
				return
			}
			defer os.RemoveAll(workDir)
			for index := range jobs {
				result, err := runOracle(qizmo, workDir, cfg.artifacts, scenarios[index])
				if err != nil {
					reportError(errCh, err)
					continue
				}
				results[index] = result
				done := completed.Add(1)
				if done%100 == 0 || done == int64(len(scenarios)) {
					fmt.Fprintf(os.Stderr, "oracle %d/%d\n", done, len(scenarios))
				}
			}
		}()
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
	}

	oracle.SortResults(results)
	data, err := oracle.MarshalManifest(oracle.NewManifest(results))
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfg.output, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %d scenarios to %s\n", len(results), cfg.output)
	return nil
}

func selectScenarios(scenarios []oracle.Scenario, family, name string) []oracle.Scenario {
	selected := scenarios[:0]
	for _, scenario := range scenarios {
		if family != "" && scenario.Family != family {
			continue
		}
		if name != "" && scenario.Name != name {
			continue
		}
		selected = append(selected, scenario)
	}
	return selected
}

func prepareWorkDir(compressDat []byte) (string, error) {
	workDir, err := os.MkdirTemp("", "qizmo-oracle-")
	if err != nil {
		return "", err
	}
	files := []struct {
		name string
		data []byte
	}{
		{"compress.dat", compressDat},
		{"qizmo.cfg", []byte("public 0\nquakedir ./\n")},
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(workDir, file.name), file.data, 0o644); err != nil {
			_ = os.RemoveAll(workDir)
			return "", err
		}
	}
	return workDir, nil
}

func runOracle(qizmo, workDir, artifacts string, scenario oracle.Scenario) (oracle.Result, error) {
	inputPath := filepath.Join(workDir, "input.qwd")
	compressedPath := filepath.Join(workDir, "input.qwz")
	decodeInputPath := filepath.Join(workDir, "decoded.qwz")
	decodedPath := filepath.Join(workDir, "decoded.qwd")
	for _, path := range []string{inputPath, compressedPath, decodeInputPath, decodedPath} {
		_ = os.Remove(path)
	}
	if err := os.WriteFile(inputPath, scenario.QWD, 0o644); err != nil {
		return oracle.Result{}, err
	}
	if err := invoke(qizmo, workDir, "-C", filepath.Base(inputPath)); err != nil {
		return oracle.Result{}, fmt.Errorf("%s compress: %w", scenario.Name, err)
	}
	compressed, err := os.ReadFile(compressedPath)
	if err != nil {
		return oracle.Result{}, fmt.Errorf("%s read compressed output: %w", scenario.Name, err)
	}
	if err := os.WriteFile(decodeInputPath, compressed, 0o644); err != nil {
		return oracle.Result{}, err
	}
	if err := invoke(qizmo, workDir, "-D", filepath.Base(decodeInputPath)); err != nil {
		return oracle.Result{}, fmt.Errorf("%s decompress: %w", scenario.Name, err)
	}
	decoded, err := os.ReadFile(decodedPath)
	if err != nil {
		return oracle.Result{}, fmt.Errorf("%s read decompressed output: %w", scenario.Name, err)
	}
	if artifacts != "" {
		name := strings.NewReplacer("/", "_", "\\", "_").Replace(scenario.Name)
		for extension, data := range map[string][]byte{"input.qwd": scenario.QWD, "qizmo.qwz": compressed, "qizmo.qwd": decoded} {
			if err := os.WriteFile(filepath.Join(artifacts, name+"."+extension), data, 0o644); err != nil {
				return oracle.Result{}, err
			}
		}
	}
	return oracle.Result{
		Name:          scenario.Name,
		Family:        scenario.Family,
		InputSize:     len(scenario.QWD),
		InputSHA256:   oracle.SHA256(scenario.QWD),
		QWZSize:       len(compressed),
		QWZSHA256:     oracle.SHA256(compressed),
		DecodedSize:   len(decoded),
		DecodedSHA256: oracle.SHA256(decoded),
	}, nil
}

func invoke(qizmo, workDir, operation, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, qizmo, "-u", "-3", operation, name)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return fmt.Errorf("timed out: %w", ctx.Err())
	}
	if err != nil {
		return fmt.Errorf("%v: %w", string(output), err)
	}
	return nil
}

func reportError(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}
