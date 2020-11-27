package db

import (
	"crypto/sha512"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"log"
	"sort"
)

// MakeBundles set bundles in the DB.
// This function changes the database!
func (db *Db) MakeBundles(keyFile *enc.KeyFile, debugLvl uint8) {
	// debug (0=off, 1=debug, 2=high)
	debug := debugLvl >= impl.DebugLow

	// reset old bundles
	resetOldBundles(db)

	// groups: [][]VirtFile
	for _, group := range findGroups(db, debug) {

		// calc bundle PlainHash  (hash all PlainSHA512)
		storageSize := int64(0)
		hh := sha512.New()
		for _, vFile := range group {
			storageSize += vFile.Parts[0].StorageSize
			hh.Write(vFile.Parts[0].PlainSHA512)
		}
		plainHash := hh.Sum(nil)

		// get bundle content (all VirtFile IDs)
		content := make([]string, 0, len(group))
		for _, virtFile := range group {
			content = append(content, virtFile.Id())
		}

		// build bundle
		bundle := Bundle{
			VFilePart: VFilePart{
				PlainSHA512:  plainHash,
				StorageName:  BundlePrefix + keyFile.CryptName(plainHash),
				StorageSize:  storageSize,
				StorageMd5:   "", // not used
				CryptDataKey: keyFile.DataKey(plainHash),
			},
			Content: content,
		}
		db.Bundles[bundle.Id()] = bundle // add bundle to db

		// connect all virtual files with the bundle
		for _, relPath := range bundle.Content {
			tmp := db.VFiles[relPath]
			tmp.AlsoInBundle = bundle.Id()
			db.VFiles[relPath] = tmp
		}
	}
}

// ----------  HELPER  -----------------------------------------------------------------------------------------------//

// resetOldBundles remove all bundles and bundle links.
func resetOldBundles(db *Db) {
	// reset virtual files
	for k := range db.VFiles {
		tmp := db.VFiles[k]
		tmp.AlsoInBundle = ""
		db.VFiles[k] = tmp
	}
	// reset bundles
	db.Bundles = make(map[string]Bundle)
}

// findGroups bundles small files to one large file
// There are NO db changes!
func findGroups(db *Db, debug bool) [][]VirtFile {
	// This is a list of bundles.
	// A bundle is a collection of virtual files.
	foundGroups := make([][]VirtFile, 0)

	// First, all files are extracted from the database that are suitable for a bundle.
	// (Files that are not empty and smaller than MaxFileSizeToBundle)
	files := make([]VirtFile, 0, len(db.VFiles))
	for _, dbEl := range db.VFiles {
		if !dbEl.IsDir && dbEl.FileSize < MaxFileSizeToBundle && dbEl.FileSize > 0 && len(dbEl.Parts) == 1 {
			files = append(files, dbEl)
		}
	}

	// To make the bundles stable (multiple calls of this function
	// always return the same result), all files are sorted by path.
	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	// Phase 1: Combine all very small text files.
	// The criterion here is the size on storage (<12 kB).
	//    MaxFileSizeToBundle > MaxFileSizeForCompression > 12 kB > [TARGET] > 0
	smallFiles := make([]VirtFile, 0, len(files)) // phase 1 list
	rest := make([]VirtFile, 0, len(files))       // left elements

	for _, v := range files {
		if v.Parts[0].StorageSize < 12*1024 { // StorageSize == compressed size
			smallFiles = append(smallFiles, v)
		} else {
			rest = append(rest, v)
		}
	}

	if debug {
		size := float64(0)
		for _, v := range smallFiles {
			size += float64(v.FileSize) / (1024 * 1024)
		}
		log.Printf("DEBUG: %s/MakeBundles: Phase 1 (very small files): bundle %d files (%.0f MB)", packageName, len(smallFiles), size)
	}
	files = rest
	foundGroups = append(foundGroups, smallFiles)

	// Phase 2: All remaining compressible files are combined from the remaining files.
	// Note: Due to the pre-filtering (MaxFileSizeToBundle) only small files are processed.
	//    MaxFileSizeToBundle > MaxFileSizeForCompression > [TARGET] > 12 kB
	mediumFiles := make([]VirtFile, 0, len(files)) // phase 2 list
	rest = make([]VirtFile, 0, len(files))         // left elements

	for _, v := range files {
		if v.UseCompression {
			mediumFiles = append(mediumFiles, v)
		} else {
			rest = append(rest, v)
		}
	}

	if debug {
		size := float64(0)
		for _, v := range smallFiles {
			size += float64(v.FileSize) / (1024 * 1024)
		}
		log.Printf("DEBUG: %s/MakeBundles: Phase 2 (compressible files): bundle %d files (%.0f MB)", packageName, len(mediumFiles), size)
	}
	files = rest
	foundGroups = append(foundGroups, mediumFiles)

	// Phase 3: Now there should be a lot of files left. Media files such as photos in particular
	// are difficult to compress and have not yet been processed.
	//
	// The algorithm works as follows:
	// The basis for the grouping is the path.
	// The maximum path length is shortened in a loop.
	// Groups are created from shortened paths with the same name.
	// If a group is large enough, it is saved as valid.
	cutAt := 0
	// find max. path len
	for _, v := range files {
		if len(v.RelPath) > cutAt {
			cutAt = len(v.RelPath)
		}
	}

	for cutAt > 0 {
		// Reduce the maximum path length by 1
		cutAt--

		// Find the total size of each group in this round.
		groups := make(map[string]int64)
		for _, v := range files {
			group := v.RelPath
			if len(group) > cutAt {
				group = group[:cutAt]
			}
			groups[group] += v.FileSize
		}

		// For all groups from a certain size:
		// Save the group and remove the files from the files list.
		for group, gSize := range groups {
			if gSize > PartSize/2 || cutAt == 0 {
				//----------------------------------------
				newG := make([]VirtFile, 0, len(files))
				rest = make([]VirtFile, 0, len(files))
				for _, v := range files {
					g := v.RelPath
					if len(g) > cutAt {
						g = g[:cutAt]
					}
					if g == group {
						newG = append(newG, v)
					} else {
						rest = append(rest, v)
					}
				}

				if debug {
					size := float64(gSize) / (1024 * 1024)
					log.Printf("DEBUG: %s/MakeBundles: Phase 3: bundle %d files (%.0f MB) with group path '%s'", packageName, len(newG), size, group)
				}
				files = rest
				foundGroups = append(foundGroups, newG)
				//----------------------------------------
			}
		}
	}

	// hotfix: split too big bundles
	foundGroups = splitBigBundles(foundGroups, debug)

	// hotfix: remove empty groups or too small groups
	tmp := make([][]VirtFile, 0)
	for i, group := range foundGroups {
		if len(group) > 1 {
			tmp = append(tmp, group)
		} else {
			if debug {
				log.Printf("DEBUG: %s/MakeBundles: discard bundle[%d] with %d files", packageName, i, len(group))
			}
		}
	}
	foundGroups = tmp

	// return
	return foundGroups
}

// splitBigBundles split bundles > 2*PartSize into PartSize sub-bundles
func splitBigBundles(foundGroups [][]VirtFile, debug bool) [][]VirtFile {

	// count all files (for test later)
	inFileCount := 0
	for _, g := range foundGroups {
		inFileCount += len(g)
	}

	// do your thing
	ret := make([][]VirtFile, 0)
	for i, group := range foundGroups {

		// calc group storage size
		size := int64(0)
		for _, f := range group {
			size += f.Parts[0].StorageSize // 'len(dbEl.Parts) == 1' is enforced by first check in parent function
		}

		// bundle size is ok
		if size <= 2*PartSize {
			ret = append(ret, group) // add ok bundle
			continue
		}

		// bundle is too big
		sub := 0
		newGroup := make([]VirtFile, 0)
		newSize := int64(0)
		for _, f := range group {
			newGroup = append(newGroup, f)
			newSize += f.Parts[0].StorageSize
			if newSize > PartSize {
				ret = append(ret, newGroup)    // add new group
				sub++                          // count new sub-bundle
				newGroup = make([]VirtFile, 0) // reset
				newSize = int64(0)             // reset
			}
		}
		ret = append(ret, newGroup) // add rest
		sub++                       // count rest sub-bundle

		if debug {
			log.Printf("DEBUG: %s/splitBigBundles: split bundle[%d] with %d files and %d MB into %d sub-bundle", packageName, i, len(group), size/(1024*1024), sub)
		}
	}

	// final test
	outFileCount := 0
	for _, g := range ret {
		outFileCount += len(g)
	}
	if inFileCount != outFileCount {
		log.Printf("ERROR: %s/splitBigBundles: inFileCount=%d, outFileCount=%d", packageName, inFileCount, outFileCount)
	}

	// return
	return ret
}
