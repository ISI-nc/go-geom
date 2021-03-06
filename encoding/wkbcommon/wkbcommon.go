// Package wkbcommon contains code common to WKB and EWKB encoding.
package wkbcommon

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Byte order IDs.
const (
	XDRID = 0
	NDRID = 1
)

// Byte orders.
var (
	XDR = binary.BigEndian
	NDR = binary.LittleEndian
)

// An ErrUnknownByteOrder is returned when an unknown byte order is encountered.
type ErrUnknownByteOrder byte

func (e ErrUnknownByteOrder) Error() string {
	return fmt.Sprintf("wkb: unknown byte order: %b", byte(e))
}

// An ErrUnsupportedByteOrder is returned when an unsupported byte order is encountered.
type ErrUnsupportedByteOrder struct{}

func (e ErrUnsupportedByteOrder) Error() string {
	return "wkb: unsupported byte order"
}

// A Type is a WKB code.
type Type uint32

// An ErrUnknownType is returned when an unknown type is encountered.
type ErrUnknownType Type

func (e ErrUnknownType) Error() string {
	return fmt.Sprintf("wkb: unknown type: %d", uint(e))
}

// An ErrUnsupportedType is returned when an unsupported type is encountered.
type ErrUnsupportedType Type

func (e ErrUnsupportedType) Error() string {
	return fmt.Sprintf("wkb: unsupported type: %d", uint(e))
}

// An ErrUnexpectedType is returned when an unexpected type is encountered.
type ErrUnexpectedType struct {
	Got  interface{}
	Want interface{}
}

func (e ErrUnexpectedType) Error() string {
	return fmt.Sprintf("wkb: got %T, want %T", e.Got, e.Want)
}

// MaxGeometryElements is the maximum number of elements that will be decoded
// at different levels. Its primary purpose is to prevent corrupt inputs from
// causing excessive memory allocations (which could be used as a denial of
// service attack).
// FIXME This should be Codec-specific, not global
// FIXME Consider overall per-geometry limit rather than per-level limit
var MaxGeometryElements = [4]uint32{
	0,
	1 << 20, // No LineString, LinearRing, or MultiPoint should contain more than 1048576 coordinates
	1 << 15, // No MultiLineString or Polygon should contain more than 32768 LineStrings or LinearRings
	1 << 10, // No MultiPolygon should contain more than 1024 Polygons
}

// An ErrGeometryTooLarge is returned when the geometry is too large.
type ErrGeometryTooLarge struct {
	Level int
	N     uint32
	Limit uint32
}

func (e ErrGeometryTooLarge) Error() string {
	return fmt.Sprintf("wkb: number of elements at level %d (%d) exceeds %d", e.Level, e.N, e.Limit)
}

// Geometry type IDs.
const (
	PointID              = 1
	LineStringID         = 2
	PolygonID            = 3
	MultiPointID         = 4
	MultiLineStringID    = 5
	MultiPolygonID       = 6
	GeometryCollectionID = 7
	PolyhedralSurfaceID  = 15
	TINID                = 16
	TriangleID           = 17
)

// ReadFlatCoords0 reads flat coordinates 0.
func ReadFlatCoords0(r io.Reader, byteOrder binary.ByteOrder, stride int) ([]float64, error) {
	coord := make([]float64, stride)
	if err := binary.Read(r, byteOrder, &coord); err != nil {
		return nil, err
	}
	return coord, nil
}

// ReadFlatCoords1 reads flat coordinates 1.
func ReadFlatCoords1(r io.Reader, byteOrder binary.ByteOrder, stride int) ([]float64, error) {
	var n uint32
	if err := binary.Read(r, byteOrder, &n); err != nil {
		return nil, err
	}
	if n > MaxGeometryElements[1] {
		return nil, ErrGeometryTooLarge{Level: 1, N: n, Limit: MaxGeometryElements[1]}
	}
	flatCoords := make([]float64, int(n)*stride)
	if err := binary.Read(r, byteOrder, &flatCoords); err != nil {
		return nil, err
	}
	return flatCoords, nil
}

// ReadFlatCoords2 reads flat coordinates 2.
func ReadFlatCoords2(r io.Reader, byteOrder binary.ByteOrder, stride int) ([]float64, []int, error) {
	var n uint32
	if err := binary.Read(r, byteOrder, &n); err != nil {
		return nil, nil, err
	}
	if n > MaxGeometryElements[2] {
		return nil, nil, ErrGeometryTooLarge{Level: 2, N: n, Limit: MaxGeometryElements[2]}
	}
	var flatCoordss []float64
	var ends []int
	for i := 0; i < int(n); i++ {
		flatCoords, err := ReadFlatCoords1(r, byteOrder, stride)
		if err != nil {
			return nil, nil, err
		}
		flatCoordss = append(flatCoordss, flatCoords...)
		ends = append(ends, len(flatCoordss))
	}
	return flatCoordss, ends, nil
}

// WriteFlatCoords0 writes flat coordinates 0.
func WriteFlatCoords0(w io.Writer, byteOrder binary.ByteOrder, coord []float64) error {
	return binary.Write(w, byteOrder, coord)
}

// WriteFlatCoords1 writes flat coordinates 1.
func WriteFlatCoords1(w io.Writer, byteOrder binary.ByteOrder, coords []float64, stride int) error {
	if err := binary.Write(w, byteOrder, uint32(len(coords)/stride)); err != nil {
		return err
	}
	return binary.Write(w, byteOrder, coords)
}

// WriteFlatCoords2 writes flat coordinates 2.
func WriteFlatCoords2(w io.Writer, byteOrder binary.ByteOrder, flatCoords []float64, ends []int, stride int) error {
	if err := binary.Write(w, byteOrder, uint32(len(ends))); err != nil {
		return err
	}
	offset := 0
	for _, end := range ends {
		if err := WriteFlatCoords1(w, byteOrder, flatCoords[offset:end], stride); err != nil {
			return err
		}
		offset = end
	}
	return nil
}
