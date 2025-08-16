package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/osm/quake/common/pak"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <input directory> <pak file>\n", os.Args[0])
		os.Exit(1)
	}

	inputDirPath := os.Args[1]
	pakFilePath := os.Args[2]

	var pakFile pak.Pak

	err := filepath.WalkDir(inputDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to walk %s: %v\n", inputDirPath, err)
			os.Exit(1)
		}

		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to stat %s: %v\n", path, err)
			os.Exit(1)
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", path, err)
			os.Exit(1)
		}

		relPath, err := filepath.Rel(inputDirPath, path)
		if err != nil {
			return err
		}

		pakFile.Files = append(pakFile.Files, &pak.File{
			Data: data,
			Path: relPath,
		})

		log.Printf("added %s (%d bytes) to %s", relPath, len(data), pakFilePath)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to walk dir (%s): %v\n", inputDirPath, err)
		os.Exit(1)
	}

	filePath := filepath.Join(pakFilePath)
	if err := ioutil.WriteFile(filePath, pakFile.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "unable to create file %s, %v", filePath, err)
		os.Exit(1)
	}
}
