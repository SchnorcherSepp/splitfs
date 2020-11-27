package enc

import (
	"errors"
	"io"
)

var _ io.ReadCloser = (*_CryptoReader)(nil)

type _CryptoReader struct {
	key    []byte
	inner  io.ReadCloser
	offset int64
}

// CryptoReader decrypts or encrypts a given reader.
// cryptOff is for the correct encryption position.
func CryptoReader(r io.ReadCloser, cryptOff int64, dataKey []byte) io.ReadCloser {
	return &_CryptoReader{
		key:    dataKey,
		inner:  r,
		offset: cryptOff,
	}
}

//--------------------------------------------------------------------------------------------------------------------//

func (cr *_CryptoReader) Read(p []byte) (n int, err error) {
	// nil reader check
	if cr.inner == nil {
		return 0, errors.New("inner reader is nil")
	}

	// read
	n, err = cr.inner.Read(p)

	// crypt and update offset
	CryptBytes(p[:n], cr.offset, cr.key)
	cr.offset += int64(n)

	// return n AND error
	return n, err
}

func (cr *_CryptoReader) Close() (err error) {
	if cr.inner != nil {
		err = cr.inner.Close()
	}
	return
}
