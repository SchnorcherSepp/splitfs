package db

import (
	"path"
)

// VirtFile stands for a single file on the local disk or on the virtual file system.
//   Db -> VirtFile -> VFilePart
type VirtFile struct {

	// --------- VirtFile description ------------------------------------------

	// RelPath is the path of the file with the mount folder as root.
	// Example: foo/bar/test.txt
	RelPath string

	// FileSize is the file size in bytes. (real local file)
	// Example 16317
	FileSize int64

	// MTime show the last change or update of the object (unix time; seconds).
	// If a file has never been changed, it's the time of creation.
	// Example: 1584535538
	MTime int64

	// IsDir marks this element as a folder.
	IsDir bool

	// --------- folder data (IsDir=true) --------------------------------------

	// FolderContent (IF FOLDER) is the list of folder sub elements.
	// Only IsDir and RelPath are set for a folder sub element.
	FolderContent []FolderEl

	// --------- file data (IsDir=false) ----------------------------------------

	// Parts (IF FILE) is the list of parts that make up the virtual file.
	// If the file size is 0, there are no parts.
	Parts []VFilePart

	// UseCompression (IF FILE) determines whether the data is compressed or not.
	// Only very small files with one part can be compressed (@see MaxFileSizeForCompression).
	// In this case, FileSize and StorageSize are different.
	// Example: true
	UseCompression bool

	// AlsoInBundle (IF FILE; OPTIONAL) is the bundle ID (@see Db.Bundles).
	// Only very small files with one part can be bundled (@see MaxFileSizeToBundle).
	AlsoInBundle string
}

// --------- more VirtFile description -----------------------------------------

// Id uniquely identifies a file (= RelPath).
func (vf *VirtFile) Id() string {
	return vf.RelPath
}

// Name of the file. The name is not unique and there can be multiple files with the same name.
func (vf *VirtFile) Name() string {
	return path.Base(vf.RelPath)
}

// --------- FolderContentEl description -----------------------------------------

// FolderEl is the list of folder sub elements.
type FolderEl struct {

	// RelPath is the path of the file with the mount folder as root.
	// Example: foo/bar/test.txt
	RelPath string

	// IsDir marks this element as a folder.
	IsDir bool
}
