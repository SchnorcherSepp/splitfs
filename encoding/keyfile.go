package enc

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"io"
	"io/ioutil"
	"os"
)

// KeyFile manages the secret keys
type KeyFile struct {
	cryptSecret []byte // for data encryption
	hashSecret  []byte // for filename encryption
	indexSecret []byte // for db encryption
}

// LoadKeyFile read the 128 bytes key file and generate the secrets.
func LoadKeyFile(path string) (*KeyFile, error) {

	// read key file
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// file size == 128 bytes
	if len(b) != 128 {
		return nil, errors.New("key file must be exactly 128 bytes long")
	}

	// keys:
	//   cryptSecret the first 64 bytes,
	//   hashSecret the last 64 bytes and
	//   indexSecret between.
	k := new(KeyFile)
	k.cryptSecret = pbkdf2.Key(b[:64], []byte("master_secret"), 60000, 64, sha512.New)
	k.hashSecret = pbkdf2.Key(b[64:], []byte("hash_secret"), 60000, 64, sha512.New)
	k.indexSecret = pbkdf2.Key(b[32:96], []byte("index_secret"), 99999, 64, sha512.New)
	return k, nil
}

// DataKey calculates the key for data.
// The key is derived from the unencrypted original data (plain SHA512).
// return 32 bytes (AES 256 key)
func (k *KeyFile) DataKey(plainHash []byte) []byte {
	return pbkdf2.Key(k.cryptSecret, plainHash, 10000, 32, sha256.New)
}

// IndexKey calculates the key for the database (index).
// return 32 bytes (AES 256 key)
func (k *KeyFile) IndexKey() []byte {
	return pbkdf2.Key(k.indexSecret, []byte("IndexKey"), 5000, 32, sha256.New)
}

// CryptName calculates the encrypted file name of a file part.
// The plain text hash of the data is required for the calculation.
// return 64 bytes (SHA 512) as hex string
func (k *KeyFile) CryptName(plainHash []byte) string {
	key := pbkdf2.Key(k.hashSecret, plainHash, 500, 64, sha512.New)
	return fmt.Sprintf("%x", key)
}

//--------------------------------------------------------------------------------------------------------------------//

// CreateKeyFile creates a new key file that contains exactly 128 random bytes.
// Existing files are NOT overwritten.
func CreateKeyFile(path string) error {
	// random key
	randKey := make([]byte, 128)
	n, err := io.ReadFull(rand.Reader, randKey)
	if err != nil {
		return err
	}
	if n != 128 || len(randKey) != 128 {
		return errors.New("can't create 128 byte key")
	}

	// don't overwrite files
	if _, err := os.Stat(path); err == nil {
		return errors.New("file already exists")
	}

	// write key file
	err = ioutil.WriteFile(path, randKey, 0600)
	if err != nil {
		return err
	}

	// read test
	k, err := LoadKeyFile(path)
	if err != nil {
		return err
	}
	k.IndexKey() // get key

	// success
	return nil
}
