package reader

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type EPUBContainer struct {
	XMLName   xml.Name `xml:"container"`
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type EPUBPackage struct {
	XMLName  xml.Name `xml:"package"`
	Manifest []struct {
		ID   string `xml:"id,attr"`
		Href string `xml:"href,attr"`
	} `xml:"manifest>item"`
	Spine []struct {
		IDRef string `xml:"idref,attr"`
	} `xml:"spine>itemref"`
}

func GetEPUBOrderedPaths(r *zip.ReadCloser) ([]string, error) {
	// 1. Find META-INF/container.xml and parse it to locate the OPF file.
	var opfPath string
	for _, f := range r.File {
		if filepath.ToSlash(strings.ToLower(f.Name)) == "meta-inf/container.xml" {
			rc, err := f.Open()
			if err != nil {
				break
			}
			var container EPUBContainer
			err = xml.NewDecoder(rc).Decode(&container)
			rc.Close()
			if err == nil && len(container.Rootfiles) > 0 {
				opfPath = container.Rootfiles[0].FullPath
			}
			break
		}
	}

	// Fallback: search for any .opf file in the ZIP.
	if opfPath == "" {
		for _, f := range r.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
				opfPath = f.Name
				break
			}
		}
	}

	if opfPath == "" {
		return nil, fmt.Errorf("no OPF package file found in EPUB")
	}

	// 2. Open and parse the OPF file.
	var opfFile *zip.File
	normOpfPath := filepath.ToSlash(strings.ToLower(opfPath))
	for _, f := range r.File {
		if filepath.ToSlash(strings.ToLower(f.Name)) == normOpfPath {
			opfFile = f
			break
		}
	}

	if opfFile == nil {
		return nil, fmt.Errorf("OPF file %s not found in ZIP", opfPath)
	}

	rc, err := opfFile.Open()
	if err != nil {
		return nil, err
	}
	var pkg EPUBPackage
	err = xml.NewDecoder(rc).Decode(&pkg)
	rc.Close()
	if err != nil {
		return nil, err
	}

	// 3. Build ID-to-Href map and parse ordered items from spine.
	manifestMap := make(map[string]string)
	for _, item := range pkg.Manifest {
		manifestMap[item.ID] = item.Href
	}

	opfDir := filepath.Dir(opfPath)
	if opfDir == "." {
		opfDir = ""
	}

	var orderedPaths []string
	for _, itemref := range pkg.Spine {
		href, exists := manifestMap[itemref.IDRef]
		if !exists {
			continue
		}

		// Clean up fragment identifiers
		if idx := strings.Index(href, "#"); idx != -1 {
			href = href[:idx]
		}

		// Unescape URL entities (like %20 -> space)
		if decoded, err := url.PathUnescape(href); err == nil {
			href = decoded
		}

		// Resolve path relative to OPF directory
		var fullHrefPath string
		if opfDir != "" {
			fullHrefPath = filepath.ToSlash(filepath.Clean(filepath.Join(opfDir, href)))
		} else {
			fullHrefPath = filepath.ToSlash(filepath.Clean(href))
		}

		orderedPaths = append(orderedPaths, fullHrefPath)
	}

	return orderedPaths, nil
}
