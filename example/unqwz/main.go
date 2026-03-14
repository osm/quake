package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/demo/qwz/assets"
	"github.com/osm/quake/demo/qwz/freq"
)

func main() {
	if len(os.Args) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s [demo.qwz]\n", os.Args[0])
		os.Exit(1)
	}

	qwzData, err := readInput()
	if err != nil {
		if len(os.Args) == 2 {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", os.Args[1], err)
		} else {
			fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
		}
		os.Exit(1)
	}

	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load embedded compress data: %v\n", err)
		os.Exit(1)
	}

	da := assets.Assets{
		PrecacheModels:     assets.PrecacheModels,
		PrecacheSounds:     assets.PrecacheSounds,
		CenterPrintStrings: assets.EmbeddedStringTable(assets.CenterPrintStrings),
		PrintMode3Strings:  assets.EmbeddedStringTable(assets.PrintMode3Strings),
		PrintStrings:       assets.EmbeddedStringTable(assets.PrintStrings),
		SetInfoStrings:     assets.EmbeddedStringTable(assets.SetInfoStrings),
		StuffTextStrings:   assets.EmbeddedStringTable(assets.StuffTextStrings),
	}

	qwdData, err := qwz.Decode(qwzData, ft, da)
	if err != nil {
		if len(os.Args) == 2 {
			fmt.Fprintf(os.Stderr, "decode %s: %v\n", os.Args[1], err)
		} else {
			fmt.Fprintf(os.Stderr, "decode stdin: %v\n", err)
		}
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		if _, err := os.Stdout.Write(qwdData); err != nil {
			fmt.Fprintf(os.Stderr, "write stdout: %v\n", err)
			os.Exit(1)
		}
		return
	}

	inputPath := os.Args[1]
	outputPath := strings.TrimSuffix(inputPath, ".qwz") + ".qwd"
	if err := os.WriteFile(outputPath, qwdData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", outputPath, err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s\n", outputPath)
}

func readInput() ([]byte, error) {
	if len(os.Args) == 1 {
		return io.ReadAll(os.Stdin)
	}

	inputPath := os.Args[1]
	if filepath.Ext(inputPath) != ".qwz" {
		return nil, fmt.Errorf("input must be a .qwz file")
	}

	return os.ReadFile(inputPath)
}
