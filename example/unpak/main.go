package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/osm/quake/common/pak"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <pak file> <output directory>\n", os.Args[0])
		os.Exit(1)
	}

	pakFilePath := os.Args[1]
	outDirPath := os.Args[2]

	data, err := ioutil.ReadFile(pakFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open file, %v", err)
		os.Exit(1)
	}

	pak, err := pak.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to parse pak file, %v", err)
		os.Exit(1)
	}

	_, err = os.Stat(outDirPath)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s does not exist\n", outDirPath)
		os.Exit(1)
	}

	for _, f := range pak.Files {
		dirPath := filepath.Join(outDirPath, filepath.Dir(f.Path))
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			fmt.Fprintf(os.Stderr, "unable to create directory %s, %v", dirPath, err)
			os.Exit(1)
		}

		filePath := filepath.Join(outDirPath, f.Path)
		if err := ioutil.WriteFile(filePath, f.Data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "unable to create file %s, %v", filePath, err)
			os.Exit(1)
		}

		log.Printf("extracted %s (%d bytes)", f.Path, len(f.Data))
	}
}
