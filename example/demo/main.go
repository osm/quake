package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"

	"github.com/osm/quake/common/ascii"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/demo/dem"
	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/svc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <demo file>\n", os.Args[0])
		os.Exit(1)
	}
	file := os.Args[1]

	ext := filepath.Ext(file)
	if !slices.Contains([]string{".dem", ".mvd", ".qwd"}, ext) {
		fmt.Fprintf(os.Stderr, "unsupported extension \"%s\"\n", ext)
		os.Exit(1)
	}

	data, err := ioutil.ReadFile(file)
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
