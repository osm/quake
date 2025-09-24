package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/osm/quake/common/ascii"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/demo/dem"
	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/svc"
)

var validExts = []string{".dem", ".mvd", ".qwd"}

func isValidExt(ext string) bool {
	return slices.Contains(validExts, ext)
}

func readGzipFile(file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	return io.ReadAll(gzr)
}

func readDemoFile(file string) ([]byte, string, error) {
	ext := filepath.Ext(file)

	switch ext {
	case ".gz":
		innerExt := filepath.Ext(strings.TrimSuffix(file, ext))
		if !isValidExt(innerExt) {
			return nil, "", fmt.Errorf("unsupported extension %q", innerExt)
		}
		data, err := readGzipFile(file)
		if err != nil {
			return nil, "", err
		}
		return data, innerExt, nil

	default:
		if !isValidExt(ext) {
			return nil, "", fmt.Errorf("unsupported extension %q", ext)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, "", err
		}
		return data, ext, nil
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <demo file>\n", os.Args[0])
		os.Exit(1)
	}
	file := os.Args[1]

	data, ext, err := readDemoFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read file, %v", err)
		os.Exit(1)
	}

	var cmds []command.Command

	switch ext {
	case ".dem":
		cmds, err = getDemCommands(data)
	case ".mvd":
		cmds, err = getMVDCommands(data)
	case ".qwd":
		cmds, err = getQWDCommands(data)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to parse demo file, %v", err)
		os.Exit(1)
	}

	for _, cmd := range cmds {
		p, ok := cmd.(*print.Command)
		if !ok || p.String == "" {
			continue
		}

		fmt.Printf("%s", ascii.Parse(p.String))
	}
}

func getDemCommands(data []byte) ([]command.Command, error) {
	demo, err := dem.Parse(context.New(), data)
	if err != nil {
		return nil, err
	}

	var cmds []command.Command

	for _, d := range demo.Data {
		gd, ok := d.Packet.(*svc.GameData)
		if !ok {
			continue
		}

		cmds = append(cmds, gd.Commands...)
	}

	return cmds, nil
}

func getMVDCommands(data []byte) ([]command.Command, error) {
	demo, err := mvd.Parse(context.New(), data)

	if err != nil {
		return nil, err
	}

	var cmds []command.Command

	for _, d := range demo.Data {
		if d.Read == nil {
			continue
		}

		gd, ok := d.Read.Packet.(*svc.GameData)
		if !ok {
			continue
		}

		cmds = append(cmds, gd.Commands...)
	}

	return cmds, nil
}

func getQWDCommands(data []byte) ([]command.Command, error) {
	demo, err := qwd.Parse(context.New(), data)

	if err != nil {
		return nil, err
	}

	var cmds []command.Command

	for _, d := range demo.Data {
		if d.Read == nil {
			continue
		}

		gd, ok := d.Read.Packet.(*svc.GameData)
		if !ok {
			continue
		}

		cmds = append(cmds, gd.Commands...)
	}

	return cmds, nil
}
