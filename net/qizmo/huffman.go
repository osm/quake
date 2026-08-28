package qizmo

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/osm/quake/qizmo/freq"
)

// A Qizmo Huffman code stores its bit length in the low five bits and the
// code itself, most-significant bit first, in the remaining bits.
type huffmanCodes [freq.Symbols]uint32

const (
	huffmanCodeBits       = 32
	huffmanCodeLengthBits = 5
	huffmanCodeLengthMask = uint32(1<<huffmanCodeLengthBits - 1)
	huffmanCodeFirstBit   = uint32(1 << (huffmanCodeBits - 1))
	huffmanByteBits       = 8

	huffmanLeafFlag       = uint16(1 << 15)
	huffmanLeafSymbolMask = huffmanLeafFlag - 1
)

func packHuffmanCode(bits uint32, length int) uint32 {
	return bits&^huffmanCodeLengthMask | uint32(length)
}

func huffmanCodeLength(code uint32) int {
	return int(code & huffmanCodeLengthMask)
}

func huffmanCodeBit(code uint32, index int) byte {
	return byte(code >> (huffmanCodeBits - 1 - index) & 1)
}

type huffmanTables struct {
	data     []byte
	codes    map[uint32]*huffmanCodes
	decoders map[uint32]*huffmanNode
}

func newHuffmanTables(data []byte) (*huffmanTables, error) {
	if len(data) != freq.CompressDatSize {
		return nil, fmt.Errorf("invalid compress.dat size %d", len(data))
	}
	return &huffmanTables{
		data:     data,
		codes:    make(map[uint32]*huffmanCodes),
		decoders: make(map[uint32]*huffmanNode),
	}, nil
}

func (t *huffmanTables) code(model uint32, symbol byte) (uint32, error) {
	codes, err := t.modelCodes(model)
	if err != nil {
		return 0, err
	}
	return codes[symbol], nil
}

func (t *huffmanTables) modelCodes(model uint32) (*huffmanCodes, error) {
	codes, ok := t.codes[model]
	if !ok {
		var err error
		codes, err = buildHuffmanCodes(t.data, model)
		if err != nil {
			return nil, err
		}
		t.codes[model] = codes
	}
	return codes, nil
}

func (t *huffmanTables) decoder(model uint32) (*huffmanNode, error) {
	root, ok := t.decoders[model]
	if ok {
		return root, nil
	}

	codes, err := t.modelCodes(model)
	if err != nil {
		return nil, err
	}
	root = newHuffmanNode()
	for symbol, code := range codes {
		node := root
		length := huffmanCodeLength(code)
		for bitIndex := 0; bitIndex < length; bitIndex++ {
			bit := huffmanCodeBit(code, bitIndex)
			if node.child[bit] == nil {
				node.child[bit] = newHuffmanNode()
			}
			node = node.child[bit]
		}
		if node.symbol >= 0 {
			return nil, fmt.Errorf("huffman model %#x has a duplicate code", model)
		}
		node.symbol = symbol
	}
	t.decoders[model] = root
	return root, nil
}

// buildHuffmanCodes mirrors Qizmo's FUN_08095690. Its stable scan order is
// significant when two symbols have equal frequencies.
func buildHuffmanCodes(data []byte, model uint32) (*huffmanCodes, error) {
	row := freq.RowIndex(model)
	if row < 0 || row >= freq.Rows {
		return nil, fmt.Errorf("huffman model %#x is outside compress.dat", model)
	}

	var counts [freq.Symbols]uint32
	offset := row * freq.RowSize
	for i := range counts {
		counts[i] = binary.LittleEndian.Uint32(data[offset+i*4:])
	}

	// Each pair is the left and right child of an internal node. A child with
	// bit 15 set is a leaf; otherwise it indexes another pair in this array.
	var tree [freq.Symbols*2 - 2]uint16
	// Qizmo keeps these links separate from the child array. During merging a
	// link changes from zero (a leaf) to the index of the symbol's subtree.
	var links [freq.Symbols]uint16
	node := len(tree) - 1
	for merges := 0; merges < freq.Symbols-1; merges++ {
		least, second := -1, -1
		for symbol, count := range counts {
			if count == 0 {
				continue
			}
			if least < 0 || count < counts[least] {
				second = least
				least = symbol
			}
			if symbol != least && (second < 0 || count < counts[second]) {
				second = symbol
			}
		}
		if least < 0 || second < 0 {
			return nil, fmt.Errorf("huffman model %#x has fewer than %d symbols", model, freq.Symbols)
		}

		tree[node] = huffmanChild(links[:], second)
		tree[node-1] = huffmanChild(links[:], least)
		counts[least] += counts[second]
		counts[second] = 0
		links[least] = uint16(node - 1)
		node -= 2
	}

	codes := &huffmanCodes{}
	assignHuffmanChild(tree[:], codes, tree[0], packHuffmanCode(0, 1))
	assignHuffmanChild(tree[:], codes, tree[1], packHuffmanCode(huffmanCodeFirstBit, 1))
	return codes, nil
}

func huffmanChild(links []uint16, symbol int) uint16 {
	if links[symbol] == 0 {
		return uint16(symbol) | huffmanLeafFlag
	}
	return links[symbol]
}

func assignHuffmanChild(
	tree []uint16,
	codes *huffmanCodes,
	child uint16,
	code uint32,
) {
	if child&huffmanLeafFlag != 0 {
		codes[child&huffmanLeafSymbolMask] = code
		return
	}
	assignHuffmanNode(tree, codes, int(child), code)
}

func assignHuffmanNode(
	tree []uint16,
	codes *huffmanCodes,
	node int,
	code uint32,
) {
	length := huffmanCodeLength(code) + 1
	left := packHuffmanCode(code, length)
	assignHuffmanChild(tree, codes, tree[node], left)

	right := left | huffmanCodeFirstBit>>(length-1)
	assignHuffmanChild(tree, codes, tree[node+1], right)
}

type bitWriter struct {
	data []byte
	bits int
}

func (w *bitWriter) writeCode(code uint32) {
	length := huffmanCodeLength(code)
	for i := 0; i < length; i++ {
		if w.bits%huffmanByteBits == 0 {
			w.data = append(w.data, 0)
		}
		bit := huffmanCodeBit(code, i)
		w.data[len(w.data)-1] |= bit << (huffmanByteBits - 1 - w.bits%huffmanByteBits)
		w.bits++
	}
}

func (w *bitWriter) writeByte(value byte) {
	bits := uint32(value) << (huffmanCodeBits - huffmanByteBits)
	w.writeCode(packHuffmanCode(bits, huffmanByteBits))
}

func (w *bitWriter) writeBytes(data []byte) {
	for _, value := range data {
		w.writeByte(value)
	}
}

var errHuffmanEOF = errors.New("unexpected end of Qizmo Huffman stream")

type bitReader struct {
	data []byte
	bits int
}

func (r *bitReader) readBit() (byte, error) {
	if r.bits >= len(r.data)*huffmanByteBits {
		return 0, errHuffmanEOF
	}
	bit := r.data[r.bits/huffmanByteBits] >>
		(huffmanByteBits - 1 - r.bits%huffmanByteBits) & 1
	r.bits++
	return bit, nil
}

func (r *bitReader) readByte() (byte, error) {
	var value byte
	for range huffmanByteBits {
		bit, err := r.readBit()
		if err != nil {
			return 0, err
		}
		value = value<<1 | bit
	}
	return value, nil
}

func (r *bitReader) readBytes(size int) ([]byte, error) {
	data := make([]byte, size)
	for i := range data {
		value, err := r.readByte()
		if err != nil {
			return nil, err
		}
		data[i] = value
	}
	return data, nil
}

type huffmanNode struct {
	child  [2]*huffmanNode
	symbol int
}

func newHuffmanNode() *huffmanNode {
	return &huffmanNode{symbol: -1}
}

func (t *huffmanTables) readSymbol(reader *bitReader, model uint32) (byte, error) {
	node, err := t.decoder(model)
	if err != nil {
		return 0, err
	}
	for node.symbol < 0 {
		bit, err := reader.readBit()
		if err != nil {
			return 0, err
		}
		node = node.child[bit]
		if node == nil {
			return 0, fmt.Errorf("invalid code in Huffman model %#x", model)
		}
	}
	return byte(node.symbol), nil
}
