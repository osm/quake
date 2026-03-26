package main

import (
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/demo/qwz/assets"
	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/standard"
	"github.com/osm/quake/demo/qwz/state"
)

type voiceFrame struct {
	record    int
	frame     int
	timestamp float32
	seq       uint32
	payload   []byte
}

type voiceSegment struct {
	startFrame int
	endFrame   int
	startTime  float32
	endTime    float32
	frameCount int
}

type config struct {
	inputPath string
	outputDir string
}

type outputs struct {
	gsmDir       string
	gsmPath      string
	indexPath    string
	segmentsPath string
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outs, frames, err := run(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d voice frames to %s\n", len(frames), cfg.outputDir)
	fmt.Printf("wrote concatenated GSM stream to %s\n", outs.gsmPath)
	fmt.Printf(
		"convert to wav: ffmpeg -f gsm -ar 8000 -ac 1 -i %s %s\n",
		outs.gsmPath,
		filepath.Join(cfg.outputDir, "voice.wav"),
	)
}

func parseConfig() (config, error) {
	outputDir := flag.String("output", "", "output directory")
	flag.Parse()

	if *outputDir == "" {
		return config{}, fmt.Errorf("usage: %s -output <dir> [demo.qwz]", os.Args[0])
	}

	if flag.NArg() > 1 {
		return config{}, fmt.Errorf("usage: %s -output <dir> [demo.qwz]", os.Args[0])
	}

	cfg := config{outputDir: *outputDir}
	if flag.NArg() == 1 {
		cfg.inputPath = flag.Arg(0)
	}

	return cfg, nil
}

func run(cfg config) (outputs, []voiceFrame, error) {
	qwzData, err := readInput(cfg)
	if err != nil {
		return outputs{}, nil, fmt.Errorf("%s: %w", inputLabel(cfg), err)
	}

	qwdData, err := decodeQWZ(qwzData)
	if err != nil {
		return outputs{}, nil, fmt.Errorf("decode %s: %w", inputLabel(cfg), err)
	}

	frames, err := extractVoiceFrames(qwdData)
	if err != nil {
		return outputs{}, nil, fmt.Errorf("extract voice: %w", err)
	}

	if len(frames) == 0 {
		return outputs{}, nil, fmt.Errorf("no voice packets found in %s", inputLabel(cfg))
	}

	outs := outputPaths(cfg.outputDir)
	if err := writeOutputs(outs, frames); err != nil {
		return outputs{}, nil, err
	}

	return outs, frames, nil
}

func readInput(cfg config) ([]byte, error) {
	if cfg.inputPath == "" {
		return io.ReadAll(os.Stdin)
	}

	inputPath := cfg.inputPath
	if filepath.Ext(inputPath) != ".qwz" {
		return nil, fmt.Errorf("input must be a .qwz file")
	}

	return os.ReadFile(inputPath)
}

func inputLabel(cfg config) string {
	if cfg.inputPath == "" {
		return "stdin"
	}

	return cfg.inputPath
}

func outputPaths(outputDir string) outputs {
	return outputs{
		gsmDir:       filepath.Join(outputDir, "gsm"),
		gsmPath:      filepath.Join(outputDir, "voice.gsm"),
		indexPath:    filepath.Join(outputDir, "index.csv"),
		segmentsPath: filepath.Join(outputDir, "segments.csv"),
	}
}

func writeOutputs(outs outputs, frames []voiceFrame) error {
	if err := os.MkdirAll(filepath.Dir(outs.gsmPath), 0755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(outs.gsmPath), err)
	}

	if err := os.MkdirAll(outs.gsmDir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", outs.gsmDir, err)
	}

	gsmFile, err := os.Create(outs.gsmPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outs.gsmPath, err)
	}
	defer gsmFile.Close()

	indexFile, err := os.Create(outs.indexPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outs.indexPath, err)
	}
	defer indexFile.Close()

	segmentsFile, err := os.Create(outs.segmentsPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outs.segmentsPath, err)
	}
	defer segmentsFile.Close()

	indexCSV := csv.NewWriter(indexFile)
	segmentsCSV := csv.NewWriter(segmentsFile)

	if err := writeIndex(indexCSV, outs, frames, gsmFile); err != nil {
		return err
	}

	if err := writeSegments(segmentsCSV, outs.segmentsPath, frames); err != nil {
		return err
	}

	indexCSV.Flush()
	if err := indexCSV.Error(); err != nil {
		return fmt.Errorf("write %s: %w", outs.indexPath, err)
	}

	segmentsCSV.Flush()
	if err := segmentsCSV.Error(); err != nil {
		return fmt.Errorf("write %s: %w", outs.segmentsPath, err)
	}

	return nil
}

func writeIndex(
	indexCSV *csv.Writer,
	outs outputs,
	frames []voiceFrame,
	gsmFile *os.File,
) error {
	if err := indexCSV.Write([]string{
		"frame",
		"record",
		"time",
		"seq",
		"voice_header",
		"gsm_file",
	}); err != nil {
		return fmt.Errorf("write %s: %w", outs.indexPath, err)
	}

	for _, frame := range frames {
		gsmName := fmt.Sprintf("%06d.gsm", frame.frame)
		gsmPayload := frame.payload[1:]
		gsmFramePath := filepath.Join(outs.gsmDir, gsmName)
		if err := os.WriteFile(gsmFramePath, gsmPayload, 0644); err != nil {
			return fmt.Errorf("write %s: %w", gsmFramePath, err)
		}

		if _, err := gsmFile.Write(gsmPayload); err != nil {
			return fmt.Errorf("write %s: %w", outs.gsmPath, err)
		}

		if err := indexCSV.Write([]string{
			fmt.Sprintf("%d", frame.frame),
			fmt.Sprintf("%d", frame.record),
			fmt.Sprintf("%.3f", frame.timestamp),
			fmt.Sprintf("%d", frame.seq),
			fmt.Sprintf("%d", frame.payload[0]),
			filepath.Join("gsm", gsmName),
		}); err != nil {
			return fmt.Errorf("write %s: %w", outs.indexPath, err)
		}
	}

	return nil
}

func writeSegments(
	segmentsCSV *csv.Writer,
	segmentsPath string,
	frames []voiceFrame,
) error {
	if err := segmentsCSV.Write([]string{
		"start_time",
		"end_time",
		"duration",
		"frames",
		"start_frame",
		"end_frame",
	}); err != nil {
		return fmt.Errorf("write %s: %w", segmentsPath, err)
	}

	for _, segment := range buildVoiceSegments(frames, 0.20) {
		duration := segment.endTime - segment.startTime
		if err := segmentsCSV.Write([]string{
			fmt.Sprintf("%.3f", segment.startTime),
			fmt.Sprintf("%.3f", segment.endTime),
			fmt.Sprintf("%.3f", duration),
			fmt.Sprintf("%d", segment.frameCount),
			fmt.Sprintf("%d", segment.startFrame),
			fmt.Sprintf("%d", segment.endFrame),
		}); err != nil {
			return fmt.Errorf("write %s: %w", segmentsPath, err)
		}
	}

	return nil
}

func decodeQWZ(qwzData []byte) ([]byte, error) {
	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		return nil, fmt.Errorf("load embedded compress data: %w", err)
	}

	decodeAssets := assets.Assets{
		PrecacheModels:     assets.PrecacheModels,
		PrecacheSounds:     assets.PrecacheSounds,
		CenterPrintStrings: assets.EmbeddedStringTable(assets.CenterPrintStrings),
		PrintMode3Strings:  assets.EmbeddedStringTable(assets.PrintMode3Strings),
		PrintStrings:       assets.EmbeddedStringTable(assets.PrintStrings),
		SetInfoStrings:     assets.EmbeddedStringTable(assets.SetInfoStrings),
		StuffTextStrings:   assets.EmbeddedStringTable(assets.StuffTextStrings),
	}

	qwdData, err := qwz.Decode(qwzData, ft, decodeAssets)
	if err != nil {
		return nil, err
	}

	return qwdData, nil
}

func extractVoiceFrames(qwdData []byte) ([]voiceFrame, error) {
	packet := state.NewPacket(0)
	var frames []voiceFrame
	offset := 0

	for record := 0; offset < len(qwdData); record++ {
		if len(qwdData)-offset < 5 {
			return nil, fmt.Errorf("truncated record header at %d", offset)
		}

		timestamp := math.Float32frombits(
			binary.LittleEndian.Uint32(qwdData[offset : offset+4]),
		)
		offset += 4

		recordType := qwdData[offset]
		offset++

		switch recordType {
		case 0x00:
			if len(qwdData)-offset < 0x24 {
				return nil, fmt.Errorf("truncated DEMO_CMD at record %d", record)
			}
			offset += 0x24
		case 0x01:
			if len(qwdData)-offset < 4 {
				return nil, fmt.Errorf("truncated DEMO_READ size at record %d", record)
			}

			size := int(binary.LittleEndian.Uint32(qwdData[offset : offset+4]))
			offset += 4

			if len(qwdData)-offset < size {
				return nil, fmt.Errorf("truncated DEMO_READ payload at record %d", record)
			}

			payload := qwdData[offset : offset+size]
			offset += size

			if len(payload) < 8 {
				if len(payload) >= 4 &&
					binary.LittleEndian.Uint32(payload[:4]) == 0xffffffff {
					continue
				}

				return nil, fmt.Errorf(
					"short packet payload at record %d",
					record,
				)
			}

			seq := binary.LittleEndian.Uint32(payload[:4])
			if seq == 0xffffffff {
				continue
			}

			decoder := standard.New(packet)
			decoder.HandleFunc(standard.QizmoVoice, func(payload []byte) {
				frames = append(frames, voiceFrame{
					record:    record,
					frame:     len(frames),
					timestamp: timestamp,
					seq:       seq,
					payload:   payload,
				})
			})

			err := decoder.Decode(payload, seq)
			if err != nil {
				return nil, fmt.Errorf(
					"parse packet at record %d: %w",
					record,
					err,
				)
			}
		case 0x02:
			if len(qwdData)-offset < 8 {
				return nil, fmt.Errorf("truncated DEMO_SET at record %d", record)
			}
			offset += 8
		default:
			return nil, fmt.Errorf(
				"unknown qwd record type 0x%02x at record %d",
				recordType,
				record,
			)
		}
	}

	return frames, nil
}

func buildVoiceSegments(frames []voiceFrame, gapThreshold float32) []voiceSegment {
	if len(frames) == 0 {
		return nil
	}

	segments := make([]voiceSegment, 0, 16)
	current := voiceSegment{
		startFrame: frames[0].frame,
		endFrame:   frames[0].frame,
		startTime:  frames[0].timestamp,
		endTime:    frames[0].timestamp,
		frameCount: 1,
	}

	for _, frame := range frames[1:] {
		if frame.timestamp-current.endTime > gapThreshold {
			segments = append(segments, current)
			current = voiceSegment{
				startFrame: frame.frame,
				endFrame:   frame.frame,
				startTime:  frame.timestamp,
				endTime:    frame.timestamp,
				frameCount: 1,
			}
			continue
		}

		current.endFrame = frame.frame
		current.endTime = frame.timestamp
		current.frameCount++
	}

	segments = append(segments, current)
	return segments
}
