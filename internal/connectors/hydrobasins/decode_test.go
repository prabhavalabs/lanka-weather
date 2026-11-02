package hydrobasins

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
)

// A real extract: three Sri Lankan sub-basins and one Indian one from the
// published archive, cut down by scripts run outside this repository.
func TestDecodeShapefileKeepsSriLankaFromARecordedExtract(t *testing.T) {
	shp, err := os.Open("testdata/sample.shp")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shp.Close() }()
	dbf, err := os.Open("testdata/sample.dbf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dbf.Close() }()
	basins, err := DecodeShapefile(shp, dbf, Bounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(basins) != 3 {
		t.Fatalf("%d basins, want the 3 in Sri Lanka", len(basins))
	}
	for _, basin := range basins {
		if basin.ID == 0 || basin.AreaKM2 <= 0 || basin.MainBasin == 0 || len(basin.Shape) == 0 {
			t.Fatalf("basin = %+v", basin)
		}
		box := basin.Shape.Bounds()
		if box.West < Bounds.West || box.East > Bounds.East {
			t.Fatalf("basin %d lies outside Sri Lanka: %+v", basin.ID, box)
		}
	}
}

// buildShapefile writes a minimal shapefile and table in memory.
func buildShapefile(records []struct {
	id    int64
	rings [][][2]float64
}) (shp, dbf []byte) {
	var body bytes.Buffer
	for index, record := range records {
		points := 0
		for _, ring := range record.rings {
			points += len(ring)
		}
		size := 44 + len(record.rings)*4 + points*16
		_ = binary.Write(&body, binary.BigEndian, int32(index+1))
		_ = binary.Write(&body, binary.BigEndian, int32(size/2))
		west, south, east, north := 1e9, 1e9, -1e9, -1e9
		for _, ring := range record.rings {
			for _, p := range ring {
				west, east = min(west, p[0]), max(east, p[0])
				south, north = min(south, p[1]), max(north, p[1])
			}
		}
		_ = binary.Write(&body, binary.LittleEndian, int32(polygonType))
		for _, value := range []float64{west, south, east, north} {
			_ = binary.Write(&body, binary.LittleEndian, value)
		}
		_ = binary.Write(&body, binary.LittleEndian, int32(len(record.rings)))
		_ = binary.Write(&body, binary.LittleEndian, int32(points))
		offset := int32(0)
		for _, ring := range record.rings {
			_ = binary.Write(&body, binary.LittleEndian, offset)
			offset += int32(len(ring))
		}
		for _, ring := range record.rings {
			for _, p := range ring {
				_ = binary.Write(&body, binary.LittleEndian, p[0])
				_ = binary.Write(&body, binary.LittleEndian, p[1])
			}
		}
	}
	header := make([]byte, 100)
	binary.BigEndian.PutUint32(header[0:], 9994)
	binary.BigEndian.PutUint32(header[24:], uint32((100+body.Len())/2))
	binary.LittleEndian.PutUint32(header[28:], 1000)
	binary.LittleEndian.PutUint32(header[32:], polygonType)

	fields := []struct {
		name   string
		length byte
	}{{"HYBAS_ID", 13}, {"NEXT_DOWN", 13}, {"MAIN_BAS", 13}, {"SUB_AREA", 12}, {"UP_AREA", 12}}
	recordSize := 1
	var table bytes.Buffer
	for _, field := range fields {
		recordSize += int(field.length)
	}
	dbfHeader := make([]byte, 32)
	binary.LittleEndian.PutUint16(dbfHeader[8:], uint16(32+len(fields)*32+1))
	binary.LittleEndian.PutUint16(dbfHeader[10:], uint16(recordSize))
	table.Write(dbfHeader)
	for _, field := range fields {
		descriptor := make([]byte, 32)
		copy(descriptor, field.name)
		descriptor[11] = 'N'
		descriptor[16] = field.length
		table.Write(descriptor)
	}
	table.WriteByte(0x0D)
	for _, record := range records {
		table.WriteByte(' ')
		fmt.Fprintf(&table, "%13d%13d%13d%12.1f%12.1f", record.id, record.id+1, record.id, 12.5, 40.5)
	}
	return append(header, body.Bytes()...), table.Bytes()
}

func squareRing(west, south, size float64) [][2]float64 {
	return [][2]float64{{west, south}, {west, south + size}, {west + size, south + size}, {west + size, south}, {west, south}}
}

func TestDecodeShapefileFiltersByBoundsAndKeepsAttributesInStep(t *testing.T) {
	shp, dbf := buildShapefile([]struct {
		id    int64
		rings [][][2]float64
	}{
		{id: 4120000001, rings: [][][2]float64{squareRing(78.9, 9.2, 0.2)}}, // India
		{id: 4120000002, rings: [][][2]float64{squareRing(80.5, 7.0, 0.1)}},
		{id: 4120000003, rings: [][][2]float64{squareRing(79.45, 9.0, 0.2)}}, // crosses the western limit
		{id: 4120000004, rings: [][][2]float64{squareRing(81.0, 8.0, 0.1), squareRing(81.2, 8.0, 0.1)}},
	})
	basins, err := DecodeShapefile(bytes.NewReader(shp), bytes.NewReader(dbf), Bounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(basins) != 2 || basins[0].ID != 4120000002 || basins[1].ID != 4120000004 {
		t.Fatalf("basins = %+v", basins)
	}
	if basins[0].NextDown != 4120000003 || basins[0].AreaKM2 != 12.5 || basins[0].UpAreaKM2 != 40.5 {
		t.Fatalf("attributes = %+v", basins[0])
	}
	if len(basins[1].Shape) != 2 {
		t.Fatalf("a basin of two parts = %+v", basins[1].Shape)
	}
}

func TestDecodeShapefileRejectsOtherFiles(t *testing.T) {
	for _, payload := range [][]byte{nil, bytes.Repeat([]byte{0}, 100), []byte(strings.Repeat("x", 200))} {
		if _, err := DecodeShapefile(bytes.NewReader(payload), bytes.NewReader(nil), Bounds); err == nil {
			t.Fatalf("want an error for %d bytes", len(payload))
		}
	}
}

func TestDecodeArchiveFindsTheShapefile(t *testing.T) {
	shp, dbf := buildShapefile([]struct {
		id    int64
		rings [][][2]float64
	}{{id: 4120000002, rings: [][][2]float64{squareRing(80.5, 7.0, 0.1)}}})
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range map[string][]byte{"hybas_as_lev12_v1c.shp": shp, "hybas_as_lev12_v1c.dbf": dbf, "readme.txt": []byte("x")} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(file, bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	basins, err := DecodeArchive(archive, Bounds)
	if err != nil || len(basins) != 1 {
		t.Fatalf("basins %+v, err %v", basins, err)
	}
	empty, _ := zip.NewReader(bytes.NewReader(emptyZip()), int64(len(emptyZip())))
	if _, err := DecodeArchive(empty, geo.Bounds{}); err == nil {
		t.Fatal("want an error for an archive with no shapefile")
	}
}

func emptyZip() []byte {
	var buffer bytes.Buffer
	_ = zip.NewWriter(&buffer).Close()
	return buffer.Bytes()
}
