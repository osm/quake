# MVZ

Lossless compression for MVD demos, inspired by the beautiful QWZ format. MVZ
uses a range coder for structure and predicted fields, and four raw DEFLATE
streams for literal bytes. Four embedded frequency profiles share the same
codec.

The API operates on complete byte slices:

```go
func Encode(mvdData []byte) ([]byte, error)
func Decode(mvzData []byte) ([]byte, error)
func EncodeWithWorkers(mvdData []byte, workers int) ([]byte, error)
func DecodeWithWorkers(mvzData []byte, workers int) ([]byte, error)
```

`Encode` and `Decode` run sequentially. The worker variants accept a limit of
at least one. `FormatVersion` identifies the version produced by the encoder.

## Format

Fields are listed in wire order. Header integers use little endian byte order.
Sizes are in bytes.

| Part          | Field                 | Type / size |
| ------------- | --------------------- | ----------- |
| File header   | Magic (`MVZ\x1a`)     | 4 bytes     |
|               | Version               | u16         |
| Chunk header  | Type (1)              | u8          |
|               | MVD size              | u32         |
|               | CRC                   | u32         |
|               | Payload size          | u32         |
| Chunk payload | Range size            | u32         |
|               | Profile ID            | u8          |
|               | Range data            | variable    |
|               | DEFLATE encoded sizes | 4 x u32     |
|               | DEFLATE decoded sizes | 4 x u32     |
|               | DEFLATE streams       | variable    |
| End entry     | Type (0)              | u8          |

DEFLATE streams are print, stufftext, centerprint, and general, in that order.
Each has its own compression history, keeping similar text and commands
together so unrelated literal bytes do not displace useful matches.

The frequency profiles were trained on four demo eras with different recording
patterns. The encoder tries all four on the first chunk and keeps the smallest
result's profile for the rest of the demo, regardless of its recording date.

Chunks are independent and target 512 KiB without splitting MVD records.
Larger records are allowed: decoded and payload sizes must each be nonzero and
at most 16 MiB. Payload size includes both stream descriptors.

CRC32 covers each decoded chunk, not chunk ordering. The end entry is
mandatory, and trailing bytes are rejected. Each chunk stores its profile ID.
