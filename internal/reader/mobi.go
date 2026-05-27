package reader

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// PDBHeader represents the header of a Palm Database file
type PDBHeader struct {
	Name           [32]byte
	Attributes     uint16
	Version        uint16
	CreationDate   uint32
	ModDate        uint32
	BackupDate     uint32
	ModNum         uint32
	AppInfoOffset  uint32
	SortInfoOffset uint32
	Type           [4]byte
	Creator        [4]byte
	IDSeed         uint32
	NextRecordList uint32
	NumRecords     uint16
}

type RecordInfo struct {
	Offset     uint32
	Attributes uint8
	UniqueID   [3]byte
}

func ExtractMOBIParagraphs(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) < 78 {
		return "", fmt.Errorf("file too small to be a MOBI/PDB file")
	}

	reader := bytes.NewReader(data)
	var header PDBHeader
	err = binary.Read(reader, binary.BigEndian, &header)
	if err != nil {
		return "", err
	}

	numRecs := int(header.NumRecords)
	if len(data) < 78+numRecs*8 {
		return "", fmt.Errorf("file too small to contain record offsets")
	}

	records := make([]RecordInfo, numRecs)
	for i := 0; i < numRecs; i++ {
		var offset uint32
		var attrs uint8
		var uniq [3]byte

		_ = binary.Read(reader, binary.BigEndian, &offset)
		_ = binary.Read(reader, binary.BigEndian, &attrs)
		_, _ = reader.Read(uniq[:])

		records[i] = RecordInfo{
			Offset:     offset,
			Attributes: attrs,
			UniqueID:   uniq,
		}
	}

	if numRecs == 0 {
		return "", fmt.Errorf("no records found in MOBI file")
	}

	rec0Start := records[0].Offset
	var rec0End uint32
	if numRecs > 1 {
		rec0End = records[1].Offset
	} else {
		rec0End = uint32(len(data))
	}

	if int(rec0End) > len(data) || rec0Start >= rec0End {
		return "", fmt.Errorf("invalid Record 0 boundary")
	}

	rec0Data := data[rec0Start:rec0End]
	if len(rec0Data) < 12 {
		// PalmDOC compression header needs at least 12 bytes
		return "", fmt.Errorf("Record 0 too small to contain PalmDOC header")
	}

	compression := binary.BigEndian.Uint16(rec0Data[0:2])
	textRecordCount := binary.BigEndian.Uint16(rec0Data[8:10])

	var textBuilder bytes.Buffer
	
	for i := 1; i <= int(textRecordCount) && i < numRecs; i++ {
		start := records[i].Offset
		var end uint32
		if i+1 < numRecs {
			end = records[i+1].Offset
		} else {
			end = uint32(len(data))
		}

		if int(end) > len(data) || start >= end {
			continue
		}

		block := data[start:end]
		if compression == 1 {
			textBuilder.Write(block)
		} else if compression == 2 {
			decompressed, err := DecompressPalmDOC(block)
			if err == nil {
				textBuilder.Write(decompressed)
			}
		}
	}

	return StripHTMLTags(textBuilder.String()), nil
}

// DecompressPalmDOC decompresses a PalmDOC LZ77 compressed byte block
func DecompressPalmDOC(compressed []byte) ([]byte, error) {
	var out bytes.Buffer
	i := 0
	n := len(compressed)

	for i < n {
		b := compressed[i]
		i++

		if b == 0x00 {
			out.WriteByte(0x00)
		} else if b >= 0x01 && b <= 0x08 {
			for k := 0; k < int(b) && i < n; k++ {
				out.WriteByte(compressed[i])
				i++
			}
		} else if b >= 0x09 && b <= 0x7f {
			out.WriteByte(b)
		} else if b >= 0x80 && b <= 0xbf {
			if i >= n {
				break
			}
			b2 := compressed[i]
			i++

			dist := ((int(b) & 0x3F) << 5) | (int(b2) >> 3)
			length := (int(b2) & 0x07) + 3

			outBytes := out.Bytes()
			if len(outBytes) < dist || dist == 0 {
				continue
			}

			startPos := len(outBytes) - dist
			for k := 0; k < length; k++ {
				char := outBytes[startPos+(k%dist)]
				out.WriteByte(char)
				outBytes = out.Bytes()
			}
		} else {
			out.WriteByte(' ')
			out.WriteByte(b ^ 0x80)
		}
	}

	return out.Bytes(), nil
}
