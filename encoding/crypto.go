package enc

import (
	"crypto/aes"
	"crypto/cipher"
)

// CryptBytes encrypts or decrypts the bytes from a chunk
// The specified offset refers to the entire chunk as a file.
//
// ATTENTION: The values in data are changed by the function.
// In the event of an error, PANIC terminates and the data remain unchanged.
// This is only possible with the wrong key length.
//
//   encryption with AES-CTR (https://gchq.github.io/CyberChef/)
//     The nonce is static! Therefore, each chunk MUST have its own key.
//     Counter start at 0 and changes with the offset.
//     There is no padding.
//     [{"op":"AES Encrypt","args":[{"option":"Hex","string":"0101010101010...256 Bit PartKey...01010101010101"},
//     {"option":"Hex","string":"00000000000000000000000000000000"},
//     {"option":"Hex","string":""},"CTR","NoPadding","Key","Hex"]}]
func CryptBytes(data []byte, offset int64, key []byte) {

	// Calculates the AES block in which the bytes start (does not have to be the beginning of the block).
	// This block number is also the counter, since we start counting 0.
	// If the offset is fully divisible by the aes block size, then the start is also the beginning of the block.
	modulo := offset % aes.BlockSize
	ivInt := (offset - modulo) / aes.BlockSize

	// aes block number -> counter
	iv := make([]byte, aes.BlockSize)
	for i := 0; i < len(iv); i++ {
		iv[i] = byte(ivInt >> uint((15-i)*8))
	}

	// AES config
	block, err := aes.NewCipher(key)
	if err != nil {
		panic("can't crypt bytes with wrong key length")
	}
	stream := cipher.NewCTR(block, iv)

	// If we do NOT start at the beginning of the block, we first have to skip a few bytes
	if modulo != 0 {
		tmp := make([]byte, modulo)
		stream.XORKeyStream(tmp, tmp)
	}

	// encryption / decryption
	stream.XORKeyStream(data, data)
}
