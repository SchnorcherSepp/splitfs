package db

// Db manages all data to display a virtual file system.
//   Db -> VirtFile -> VFilePart
type Db struct {

	// VFiles contains all virtual files.
	// The map key is the RelPath (@see VirtFile.Id).
	VFiles map[string]VirtFile

	// Bundles is OPTIONAL and bundles some small virtual files together.
	// The map key is the StorageName of the bundle file (@see VFilePart.Id).
	Bundles map[string]Bundle
}

// Bundle is an element of Db.Bundles.
type Bundle struct {
	// VFilePart is the storage file (=one bundle)
	VFilePart // extension

	// Content is the list of VirtFile (string: VFiles map key) in this bundle.
	Content []string
}

// NewDb returns an empty database.
func NewDb() Db {
	return Db{
		VFiles:  make(map[string]VirtFile),
		Bundles: nil,
	}
}
