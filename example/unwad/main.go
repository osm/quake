package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/osm/quake/common/wad"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <wad file> <output directory>\n", os.Args[0])
		os.Exit(1)
	}

	wadFilePath := os.Args[1]
	outDirPath := os.Args[2]

	data, err := ioutil.ReadFile(wadFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open file, %v", err)
		os.Exit(1)
	}

	wad, err := wad.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to parse wad file, %v", err)
		os.Exit(1)
	}

	_, err = os.Stat(outDirPath)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s does not exist\n", outDirPath)
		os.Exit(1)
	}

	for _, e := range wad.Entries {
		lmpPath := filepath.Join(outDirPath, e.Name) + "." + e.Type.String()
		lmpData := e.Lump.Bytes()
		if err := os.WriteFile(lmpPath, lmpData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "unable to write lump %s: %v", lmpPath, err)
		}

		pngPath := filepath.Join(outDirPath, e.Name) + ".png"
		pngData, err := e.Lump.ToPNG()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to encode PNG: %v", err)
			os.Exit(1)
		}
		if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "unable to write png %s: %v", pngPath, err)
		}

		fmt.Printf("%s %s\n", e.Name, e.Type)
	}
}
