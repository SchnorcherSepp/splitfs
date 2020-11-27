package db

// packageName is used for debug and error messages
const packageName = "db"

// PartSize defines the size of the parts into which a file is split.
// It should be a multiple of the FUSE buffer (131072 bytes) and a block size of the hard drives (4096 bytes).
const PartSize = 131072 * 4096 * 2 // 1073741824 Byte (1 GB)

// MaxFileSizeForCompression specifies the maximum size of files that can be compressed.
// Attention: Every time compressed files are accessed, the server must keep them in ram!
const MaxFileSizeForCompression = 1 * 1024 * 1024 // 1 MB

// MaxFileSizeToBundle specifies the maximum files size that can be bundled together.
// Keep this number low. Only small files (e.g. pictures) should be bundled.
const MaxFileSizeToBundle = 12 * 1024 * 1024 // 12 MB

// BundlePrefix is placed in front of each bundle storage filename.
const BundlePrefix = "B_"
