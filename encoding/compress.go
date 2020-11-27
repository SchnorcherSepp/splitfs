package enc

import (
	"errors"
	"github.com/klauspost/compress/zstd"
)

// Compress use Zstandard compression algorithm to compress data.
// zstd.SpeedBestCompression is used:
// https://github.com/klauspost/compress/tree/master/zstd
//
// special case: nil input == []byte{} output
// special case: []byte{} input == []byte{} output
func Compress(in []byte) (out []byte, ratio float32, err error) {
	// no input, no output
	if in == nil || len(in) == 0 {
		return []byte{}, 1, nil
	}

	// init encoder
	optBest := zstd.WithEncoderLevel(zstd.SpeedBestCompression)
	encoder, err := zstd.NewWriter(nil, optBest)
	if err != nil {
		return []byte{}, 1, err
	}
	defer encoder.Close()

	// compression
	buf := make([]byte, 0, len(in))
	out = encoder.EncodeAll(in, buf)
	ratio = float32(len(out)) / float32(len(in))

	return out, ratio, nil
}

// Decompress use Zstandard compression algorithm to decompress data.
// https://github.com/klauspost/compress/tree/master/zstd
//
// special case: nil input == []byte{} output
// special case: []byte{} input == []byte{} output
func Decompress(in []byte) (out []byte, err error) {
	// no input, no output
	if in == nil || len(in) == 0 {
		return []byte{}, nil
	}

	// min size for magic number = 4 bytes
	if len(in) < 4 {
		err := errors.New("magic number invalid")
		return []byte{}, err
	}

	// init decoder
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return []byte{}, err
	}
	defer decoder.Close()

	// decompression
	buf := make([]byte, 0, len(in))
	return decoder.DecodeAll(in, buf)
}
