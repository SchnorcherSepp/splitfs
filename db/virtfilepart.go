package db

// VFilePart is a part of a virtual file.
//   Db -> VirtFile -> VFilePart
type VFilePart struct {

	// PlainSHA512 is the SHA512 hash of the virtual file part content (plain text).
	// This is needed for the data encryption key and the encrypted file name.
	// Example: 64 bytes (SHA512)
	PlainSHA512 []byte

	// --- this three attributes identify the storage file (@see interf.Files.ByAttr) --- //

	// StorageName of the storage file name. The name is not unique and there can be multiple files with the same name.
	// Example: f233e8122942b4dd068237ed73b123092c7e59964626f7ae842a94ad3c4cc7a9791698fa080d0d4382dea1c6a3cb6d30
	StorageName string

	// StorageSize is the storage file size in bytes.
	// Example 16317
	StorageSize int64

	// StorageMd5 is the hash of the storage file content (hex string).
	// Example: 098f6bcd4621d373c0de4e832627b4f6
	StorageMd5 string

	// --- this attribute is for the data access --- //

	// CryptDataKey is the key for encrypting and decrypting data.
	// The key is derived from the unencrypted original content (PlainSHA512).
	// Example: 32 bytes (AES256 key)
	CryptDataKey []byte
}

// Id uniquely identifies a part (= StorageName).
func (p *VFilePart) Id() string {
	return p.StorageName
}
