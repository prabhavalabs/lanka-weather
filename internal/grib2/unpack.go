package grib2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// unpack turns section 7's packed integers into values with section 5's
// template: Y = (R + X × 2^E) / 10^D, where R is the reference value, E the
// binary and D the decimal scale factor (section 5, octets 12–19).
func unpack(representation, data []byte, points int) ([]float32, error) {
	if len(representation) < 21 {
		return nil, errors.New("short data representation section")
	}
	if count := int(binary.BigEndian.Uint32(representation[5:9])); count != points {
		return nil, fmt.Errorf("%w: %d values for %d points (a bitmap?)", ErrUnsupported, count, points)
	}
	template := binary.BigEndian.Uint16(representation[9:11])
	reference := float64(math.Float32frombits(binary.BigEndian.Uint32(representation[11:15])))
	binaryScale := math.Exp2(float64(signMagnitude(uint64(binary.BigEndian.Uint16(representation[15:17])), 16)))
	decimalScale := math.Pow10(-int(signMagnitude(uint64(binary.BigEndian.Uint16(representation[17:19])), 16)))
	width := int(representation[19])

	var packed []int64
	var err error
	switch template {
	case 0:
		packed, err = unpackSimple(data, width, points)
	case 2, 3:
		packed, err = unpackComplex(representation, data, width, points, template == 3)
	default:
		return nil, fmt.Errorf("%w: data representation template 5.%d", ErrUnsupported, template)
	}
	if err != nil {
		return nil, err
	}
	values := make([]float32, points)
	for index, value := range packed {
		values[index] = float32((reference + float64(value)*binaryScale) * decimalScale)
	}
	return values, nil
}

// unpackSimple reads template 5.0: every value in `width` bits. A width of
// zero is a constant field.
func unpackSimple(data []byte, width, points int) ([]int64, error) {
	values := make([]int64, points)
	if width == 0 {
		return values, nil
	}
	reader := bitReader{data: data}
	for index := range values {
		value, err := reader.read(width)
		if err != nil {
			return nil, err
		}
		values[index] = int64(value)
	}
	return values, nil
}

// unpackComplex reads templates 5.2 and 5.3: the values are split into
// groups, each with its own reference, bit width and length. Section 7
// holds, each part starting on a whole octet:
//
//  1. for 5.3 only, the first one or two original values and the minimum
//     of the differences, in the number of octets octet 49 gives;
//  2. each group's reference, in the field's bit width (octet 20);
//  3. each group's width, in octet 37's bits, added to octet 36;
//  4. each group's length, in octet 47's bits, scaled by octet 42 and added
//     to octets 38–41; the last group's true length is octets 43–46;
//  5. the groups' values back to back, each in its group's width.
//
// With spatial differencing the result is then the first or second order
// differences of the field, less their minimum, which are summed back.
func unpackComplex(representation, data []byte, width, points int, differenced bool) ([]int64, error) {
	if len(representation) < 47 || (differenced && len(representation) < 49) {
		return nil, errors.New("short complex packing template")
	}
	if missing := representation[22]; missing != 0 {
		return nil, fmt.Errorf("%w: missing value management %d", ErrUnsupported, missing)
	}
	word := func(octet int) int { return int(binary.BigEndian.Uint32(representation[octet-1 : octet+3])) }
	groups := word(32)
	widthReference, widthBits := int(representation[35]), int(representation[36])
	lengthReference, lengthIncrement, lastLength := word(38), int(representation[41]), word(43)
	lengthBits := int(representation[46])
	if groups < 1 || groups > points {
		return nil, fmt.Errorf("%d groups for %d points", groups, points)
	}

	reader := bitReader{data: data}
	var order, first, second, minimum int64
	if differenced {
		order = int64(representation[47])
		octets := int(representation[48])
		if order != 1 && order != 2 {
			return nil, fmt.Errorf("%w: spatial differencing of order %d", ErrUnsupported, order)
		}
		if octets < 1 || octets > 4 {
			return nil, fmt.Errorf("%d octets for the extra descriptors", octets)
		}
		read := func() (int64, error) {
			value, err := reader.read(8 * octets)
			return signMagnitude(value, 8*octets), err
		}
		var err error
		if first, err = read(); err != nil {
			return nil, err
		}
		if order == 2 {
			if second, err = read(); err != nil {
				return nil, err
			}
		}
		if minimum, err = read(); err != nil {
			return nil, err
		}
	}

	references, err := reader.readAll(groups, width, 0, 1)
	if err != nil {
		return nil, err
	}
	widths, err := reader.readAll(groups, widthBits, widthReference, 1)
	if err != nil {
		return nil, err
	}
	lengths, err := reader.readAll(groups, lengthBits, lengthReference, lengthIncrement)
	if err != nil {
		return nil, err
	}
	lengths[groups-1] = lastLength
	total := 0
	for index := range groups {
		if widths[index] > 32 {
			return nil, fmt.Errorf("group %d is %d bits wide", index, widths[index])
		}
		total += lengths[index]
		if lengths[index] < 0 || total > points {
			return nil, fmt.Errorf("groups hold more than %d values", points)
		}
	}
	if total != points {
		return nil, fmt.Errorf("groups hold %d values for %d points", total, points)
	}

	values := make([]int64, 0, points)
	for index := range groups {
		reference := int64(references[index])
		for range lengths[index] {
			value := reference
			if widths[index] > 0 {
				packed, err := reader.read(widths[index])
				if err != nil {
					return nil, err
				}
				value += int64(packed)
			}
			values = append(values, value)
		}
	}

	if differenced {
		values[0] = first
		start := 1
		if order == 2 && points > 1 {
			values[1] = second
			start = 2
		}
		for index := start; index < points; index++ {
			if order == 1 {
				values[index] += minimum + values[index-1]
			} else {
				values[index] += minimum + 2*values[index-1] - values[index-2]
			}
		}
	}
	return values, nil
}

// bitReader reads big-endian unsigned integers of any width up to 64 bits.
type bitReader struct {
	data []byte
	bit  int
}

var errShortData = errors.New("data section ends early")

func (reader *bitReader) read(bits int) (uint64, error) {
	if bits == 0 {
		return 0, nil
	}
	if bits > 64 || reader.bit+bits > 8*len(reader.data) {
		return 0, errShortData
	}
	var value uint64
	for bits > 0 {
		octet := reader.data[reader.bit/8]
		used := reader.bit % 8
		take := min(8-used, bits)
		chunk := (octet >> (8 - used - take)) & (1<<take - 1)
		value = value<<take | uint64(chunk)
		reader.bit += take
		bits -= take
	}
	return value, nil
}

// readAll reads count values of `bits` each as reference + value × scale,
// then moves to the next whole octet.
func (reader *bitReader) readAll(count, bits, reference, scale int) ([]int, error) {
	values := make([]int, count)
	for index := range values {
		value, err := reader.read(bits)
		if err != nil {
			return nil, err
		}
		values[index] = reference + int(value)*scale
	}
	reader.bit = (reader.bit + 7) / 8 * 8
	return values, nil
}
