// Command mvz compresses MVD demos to MVZ and decompresses MVZ demos to MVD.
package main

import (
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/osm/quake/demo/mvz"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "mvz: %v\n", err)
		os.Exit(1)
	}
}

type config struct {
	compress  bool
	workers   int
	inputPath string
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cfg, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}
	input, inputName, outputPath, err := readInput(cfg, stdin)
	if err != nil {
		return err
	}
	output, err := transform(input, inputName, cfg.compress, cfg.workers)
	if err != nil {
		return err
	}
	return writeOutput(output, outputPath, stdout)
}

func parseFlags(args []string, stderr io.Writer) (config, error) {
	flags := flag.NewFlagSet("mvz", flag.ContinueOnError)
	flags.SetOutput(stderr)
	compress := flags.Bool("c", false, "compress MVD or gzipped MVD to MVZ")
	decompress := flags.Bool("d", false, "decompress MVZ to MVD")
	workers := flags.Int("workers", min(8, runtime.GOMAXPROCS(0)), "maximum codec workers")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: mvz (-c|-d) [-workers N] [file]\n")
		fmt.Fprintln(flags.Output())
		fmt.Fprintln(
			flags.Output(),
			"With a file, mvz writes the corresponding .mvz or .mvd file.",
		)
		fmt.Fprintln(flags.Output(), "The output file must not already exist.")
		fmt.Fprintf(flags.Output(), "Without a file, mvz reads stdin and writes stdout.\n\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if *compress == *decompress {
		flags.Usage()
		return config{}, errors.New("exactly one of -c or -d is required")
	}
	if *workers < 1 {
		return config{}, errors.New("-workers must be at least 1")
	}
	if flags.NArg() > 1 {
		flags.Usage()
		return config{}, errors.New("at most one input file may be specified")
	}
	cfg := config{compress: *compress, workers: *workers}
	if flags.NArg() == 1 {
		cfg.inputPath = flags.Arg(0)
	}
	return cfg, nil
}

func readInput(cfg config, stdin io.Reader) ([]byte, string, string, error) {
	if cfg.inputPath == "" {
		input, err := io.ReadAll(stdin)
		if err != nil {
			return nil, "", "", fmt.Errorf("read stdin: %w", err)
		}
		return input, "stdin", "", nil
	}
	outputPath, err := derivedOutputPath(cfg.inputPath, cfg.compress)
	if err != nil {
		return nil, "", "", err
	}
	var input []byte
	if cfg.compress && strings.HasSuffix(cfg.inputPath, ".mvd.gz") {
		input, err = readGzipFile(cfg.inputPath)
	} else {
		input, err = os.ReadFile(cfg.inputPath)
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("read %s: %w", cfg.inputPath, err)
	}
	return input, cfg.inputPath, outputPath, nil
}

func readGzipFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func transform(
	input []byte,
	inputName string,
	compress bool,
	workers int,
) ([]byte, error) {
	if compress {
		output, err := mvz.EncodeWithWorkers(input, workers)
		if err != nil {
			return nil, fmt.Errorf("compress %s: %w", inputName, err)
		}
		return output, nil
	}
	output, err := mvz.DecodeWithWorkers(input, workers)
	if err != nil {
		return nil, fmt.Errorf("decompress %s: %w", inputName, err)
	}
	return output, nil
}

func writeOutput(output []byte, outputPath string, stdout io.Writer) error {
	if outputPath == "" {
		if _, err := stdout.Write(output); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
		return nil
	}
	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", outputPath, err)
	}
	_, err = file.Write(output)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(outputPath)
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	if _, err := fmt.Fprintf(stdout, "wrote %s\n", outputPath); err != nil {
		return fmt.Errorf("report output: %w", err)
	}
	return nil
}

func derivedOutputPath(inputPath string, compress bool) (string, error) {
	if !compress {
		if !strings.HasSuffix(inputPath, ".mvz") {
			return "", errors.New("input file must have a .mvz extension")
		}
		return strings.TrimSuffix(inputPath, ".mvz") + ".mvd", nil
	}
	for _, suffix := range []string{".mvd", ".mvd.gz"} {
		if strings.HasSuffix(inputPath, suffix) {
			return strings.TrimSuffix(inputPath, suffix) + ".mvz", nil
		}
	}
	return "", errors.New("input file must have a .mvd or .mvd.gz extension")
}
