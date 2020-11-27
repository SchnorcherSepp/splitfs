package core

import (
	"errors"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"io"
	"log"
)

var _ interf.ReaderService = (*_CryptRService)(nil)

type _CryptRService struct {
	inner interf.ReaderService
	keys  map[string][]byte
}

// newCryptRService encapsulates a interf.ReaderService and returns all readers encrypted.
func newCryptRService(inner interf.ReaderService, keys map[string][]byte) interf.ReaderService {
	return &_CryptRService{
		inner: inner,
		keys:  keys,
	}
}

//--------------------------------------------------------------------------------------------------------------------//

func (s *_CryptRService) Reader(file interf.File, off int64) (io.ReadCloser, error) {
	return s.LimitedReader(file, off, interf.MaxFileSize) // delegate to LimitedReader
}

func (s *_CryptRService) LimitedReader(file interf.File, off int64, n int64) (io.ReadCloser, error) {
	// nil check
	if s.inner == nil || s.keys == nil || len(s.keys) == 0 || file == nil {
		return nil, errors.New("inner variables are not initialized")
	}

	// get dataKey
	dataKey, ok := s.keys[file.Id()]
	if !ok || dataKey == nil || len(dataKey) == 0 {
		e := errors.New("dataKey not found")
		log.Printf("ERROR: %s/CryptRService: %v: %s", packageName, e, file.Name())
		return nil, e
	}

	// get plain reader from inner service
	r, err := s.inner.LimitedReader(file, off, n) // INNER
	if err != nil {
		return nil, err
	}

	// return crypt reader
	return enc.CryptoReader(r, off, dataKey), err
}
