// Package hydrobasins loads the HydroBASINS level 12 sub-basins of Sri Lanka
// from HydroSHEDS' Asia-wide shapefile. They are the finer flood-forecast
// boundaries shown when zoomed in. Country-level basins come from the
// Irrigation Department instead (package irrigationgis).
package hydrobasins

import (
	"archive/zip"
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
)

// SourceName identifies the source in stored records and docs/data-sources.md.
const SourceName = "hydrosheds-hydrobasins"

// Attribution is the credit HydroSHEDS' licence asks for.
const Attribution = "HydroBASINS (Lehner & Grill 2013, HydroSHEDS)"

// Basin is one level 12 sub-basin.
type Basin struct {
	ID, NextDown, MainBasin int64
	AreaKM2, UpAreaKM2      float64
	Shape                   geo.MultiPolygon
}

// Bounds limits what is kept from the continent. Below 79.5° E and above
// 9.95° N lies India (Rameswaram, the Tamil Nadu coast); no part of Sri Lanka
// does. A sub-basin must lie wholly inside.
var Bounds = geo.Bounds{West: 79.5, South: 5.8, East: 82.0, North: 9.95}

const polygonType = 5

// maxPoints bounds a record, so a corrupt length cannot exhaust memory.
const maxPoints = 2_000_000

type dbfField struct {
	name           string
	offset, length int
}

// dbfReader reads fixed-length dBASE records one at a time.
type dbfReader struct {
	reader *bufio.Reader
	fields map[string]dbfField
	record []byte
}

func newDBFReader(reader io.Reader) (*dbfReader, error) {
	buffered := bufio.NewReaderSize(reader, 1<<16)
	header := make([]byte, 32)
	if _, err := io.ReadFull(buffered, header); err != nil {
		return nil, fmt.Errorf("dbf header: %w", err)
	}
	headerSize := int(binary.LittleEndian.Uint16(header[8:10]))
	recordSize := int(binary.LittleEndian.Uint16(header[10:12]))
	if headerSize < 33 || recordSize < 1 {
		return nil, errors.New("dbf header: implausible sizes")
	}
	rest := make([]byte, headerSize-32)
	if _, err := io.ReadFull(buffered, rest); err != nil {
		return nil, fmt.Errorf("dbf fields: %w", err)
	}
	fields := map[string]dbfField{}
	offset := 1 // the first byte of a record is its deletion flag
	for start := 0; start+32 <= len(rest) && rest[start] != 0x0D; start += 32 {
		name := strings.ToUpper(strings.TrimRight(string(rest[start:start+11]), "\x00 "))
		length := int(rest[start+16])
		fields[name] = dbfField{name: name, offset: offset, length: length}
		offset += length
	}
	return &dbfReader{reader: buffered, fields: fields, record: make([]byte, recordSize)}, nil
}

func (reader *dbfReader) next() error {
	_, err := io.ReadFull(reader.reader, reader.record)
	return err
}

func (reader *dbfReader) number(name string) (float64, error) {
	field, ok := reader.fields[name]
	if !ok || field.offset+field.length > len(reader.record) {
		return 0, fmt.Errorf("dbf has no field %s", name)
	}
	text := strings.TrimSpace(string(reader.record[field.offset : field.offset+field.length]))
	if text == "" {
		return 0, nil
	}
	return strconv.ParseFloat(text, 64)
}

// inside reports whether a record's box lies within the bounds.
func inside(west, south, east, north float64, bounds geo.Bounds) bool {
	return west >= bounds.West && east <= bounds.East && south >= bounds.South && north <= bounds.North
}

// DecodeShapefile reads a polygon shapefile and its attribute table in
// parallel, record by record, keeping those within the bounds. Other records
// are read past without being held, so a continent's file costs only the
// memory of what is kept. It does no I/O beyond its readers.
func DecodeShapefile(shp, dbf io.Reader, bounds geo.Bounds) ([]Basin, error) {
	shapes := bufio.NewReaderSize(shp, 1<<20)
	header := make([]byte, 100)
	if _, err := io.ReadFull(shapes, header); err != nil {
		return nil, fmt.Errorf("shp header: %w", err)
	}
	if code := binary.BigEndian.Uint32(header[0:4]); code != 9994 {
		return nil, fmt.Errorf("not a shapefile (code %d)", code)
	}
	if kind := binary.LittleEndian.Uint32(header[32:36]); kind != polygonType {
		return nil, fmt.Errorf("shapefile holds type %d, want polygons", kind)
	}
	attributes, err := newDBFReader(dbf)
	if err != nil {
		return nil, err
	}
	var basins []Basin
	recordHeader := make([]byte, 8)
	var content []byte
	for {
		if _, err := io.ReadFull(shapes, recordHeader); err != nil {
			if errors.Is(err, io.EOF) {
				return basins, nil
			}
			return nil, fmt.Errorf("shp record: %w", err)
		}
		if err := attributes.next(); err != nil {
			return nil, fmt.Errorf("dbf record: %w", err)
		}
		size := int(binary.BigEndian.Uint32(recordHeader[4:8])) * 2
		if size < 4 || size > maxPoints*16 {
			return nil, fmt.Errorf("shp record of %d bytes", size)
		}
		if cap(content) < size {
			content = make([]byte, size)
		}
		content = content[:size]
		if _, err := io.ReadFull(shapes, content); err != nil {
			return nil, fmt.Errorf("shp content: %w", err)
		}
		if binary.LittleEndian.Uint32(content[0:4]) != polygonType || size < 44 {
			continue // a null shape
		}
		box := func(index int) float64 {
			return math.Float64frombits(binary.LittleEndian.Uint64(content[4+index*8:]))
		}
		if !inside(box(0), box(1), box(2), box(3), bounds) {
			continue
		}
		basin, err := decodeBasin(content, attributes)
		if err != nil {
			return nil, err
		}
		if basin.ID != 0 && len(basin.Shape) > 0 {
			basins = append(basins, basin)
		}
	}
}

func decodeBasin(content []byte, attributes *dbfReader) (Basin, error) {
	parts := int(binary.LittleEndian.Uint32(content[36:40]))
	points := int(binary.LittleEndian.Uint32(content[40:44]))
	pointStart := 44 + parts*4
	if parts < 0 || points < 0 || points > maxPoints || pointStart+points*16 > len(content) {
		return Basin{}, errors.New("shp polygon: implausible part or point count")
	}
	rings := make([]geo.Ring, 0, parts)
	for part := range parts {
		from := int(binary.LittleEndian.Uint32(content[44+part*4:]))
		to := points
		if part+1 < parts {
			to = int(binary.LittleEndian.Uint32(content[44+(part+1)*4:]))
		}
		if from < 0 || to > points || from > to {
			return Basin{}, errors.New("shp polygon: implausible ring offsets")
		}
		ring := make(geo.Ring, to-from)
		for index := from; index < to; index++ {
			at := pointStart + index*16
			ring[index-from] = geo.Point{
				math.Float64frombits(binary.LittleEndian.Uint64(content[at:])),
				math.Float64frombits(binary.LittleEndian.Uint64(content[at+8:])),
			}
		}
		rings = append(rings, ring)
	}
	number := func(name string, failure *error) float64 {
		value, err := attributes.number(name)
		if err != nil && *failure == nil {
			*failure = err
		}
		return value
	}
	var failure error
	basin := Basin{
		ID: int64(number("HYBAS_ID", &failure)), NextDown: int64(number("NEXT_DOWN", &failure)),
		MainBasin: int64(number("MAIN_BAS", &failure)), AreaKM2: number("SUB_AREA", &failure),
		UpAreaKM2: number("UP_AREA", &failure), Shape: geo.PolygonFromRings(rings),
	}
	return basin, failure
}

// DecodeArchive reads the shapefile from an opened zip archive.
func DecodeArchive(archive *zip.Reader, bounds geo.Bounds) ([]Basin, error) {
	var shp, dbf *zip.File
	for _, file := range archive.File {
		switch strings.ToLower(file.Name[max(0, len(file.Name)-4):]) {
		case ".shp":
			shp = file
		case ".dbf":
			dbf = file
		}
	}
	if shp == nil || dbf == nil {
		return nil, errors.New("the archive has no .shp and .dbf")
	}
	shapes, err := shp.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = shapes.Close() }()
	attributes, err := dbf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = attributes.Close() }()
	return DecodeShapefile(shapes, attributes, bounds)
}
