package g53

import (
	"bytes"
	"errors"

	"github.com/zdnscloud/g53/util"
)

type NameRelation int

const (
	SUPERDOMAIN    NameRelation = 0
	SUBDOMAIN      NameRelation = 1
	EQUAL          NameRelation = 2
	COMMONANCESTOR NameRelation = 3
	NONE           NameRelation = 4
)

const (
	MAX_WIRE      = 255
	MAX_LABELS    = 128
	MAX_LABEL_LEN = 63

	MAX_COMPRESS_POINTER    = 0x3fff
	COMPRESS_POINTER_MARK8  = 0xc0
	COMPRESS_POINTER_MARK16 = 0xc000
)

var digitvalue = [256]int{
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 16
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 32
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 48
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, -1, -1, -1, -1, -1, -1, // 64
	-1, 10, 11, 12, 13, 14, 15, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 80
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 96
	-1, 10, 11, 12, 13, 14, 15, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 112
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 128
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 256
}

var maptolower = [256]byte{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
	0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
	0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
	0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
	0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f,
	0x40, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
	0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f,
	0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
	0x78, 0x79, 0x7a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
	0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
	0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f,
	0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
	0x78, 0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f,
	0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87,
	0x88, 0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f,
	0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97,
	0x98, 0x99, 0x9a, 0x9b, 0x9c, 0x9d, 0x9e, 0x9f,
	0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
	0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf,
	0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7,
	0xb8, 0xb9, 0xba, 0xbb, 0xbc, 0xbd, 0xbe, 0xbf,
	0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7,
	0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf,
	0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7,
	0xd8, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde, 0xdf,
	0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
	0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
	0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
	0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff}

type NameComparisonResult struct {
	Order            int
	CommonLabelCount int
	Relation         NameRelation
}

//for www.knet.cn
//{raw:			[3 119 119 119 4 107 110 101 116 2 99 110 0
// offsets: 	[0 4 9 12];label count positions
// length: 		13
// labelCount: 	4}
type Name struct {
	raw        []byte
	offsets    []byte
	length     uint
	labelCount uint
}

type ftState int

const (
	ftInit ftState = iota
	ftStart
	ftOrdinary
	ftInitialescape
	ftEscape
	ftEscdecimal
)

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func stringParse(nameRaw []byte, start uint, end uint, downcase bool) ([]byte, []byte, error) {
	data := make([]byte, 0, end-start+1)
	offsets := []byte{}
	count := 0
	digits := 0
	value := 0

	done := false
	isRoot := false
	state := ftInit

	offsets = append(offsets, 0)
	for len(data) < MAX_WIRE && start != end && done == false {
		c := nameRaw[start]
		start++
	again:
		//fmt.Printf("c is %v, pos is %v, data is %v, offsets is %v\n", string(c), start, string(data), offsets)
		switch state {
		case ftInit:
			if c == '.' {
				if start != end {
					return nil, nil, errors.New("non terminating empty label")
				}
				isRoot = true
			} else if c == '@' && start == end {
				isRoot = true
			}

			if isRoot {
				data = append(data, 0)
				done = true
				break
			}
			state = ftStart
			goto again
		case ftStart:
			data = append(data, 0)
			count = 0
			if c == '\\' {
				state = ftInitialescape
				break
			}
			state = ftOrdinary
			goto again
		case ftOrdinary:
			if c == '.' {
				if count == 0 {
					return nil, nil, errors.New("duplicate period")
				}
				data[offsets[len(offsets)-1]] = byte(count)
				offsets = append(offsets, byte(len(data)))
				if start == end {
					data = append(data, 0)
					done = true
				}
				state = ftStart
			} else if c == '\\' {
				state = ftEscape
			} else {
				count += 1
				if count > MAX_LABEL_LEN {
					return nil, nil, errors.New("too long label")
				}
				if downcase {
					data = append(data, maptolower[c])
				} else {
					data = append(data, c)
				}
			}
		case ftInitialescape:
			if c == '[' {
				return nil, nil, errors.New("invalid label type")
			}
			state = ftEscape
			goto again
		case ftEscape:
			if isDigit(c&0xff) == false {
				count += 1
				if count > MAX_LABEL_LEN {
					return nil, nil, errors.New("too long label")
				}
				if downcase {
					data = append(data, maptolower[c])
				} else {
					data = append(data, c)
				}
				state = ftOrdinary
				break
			}
			digits = 0
			value = 0
			state = ftEscdecimal
			goto again
		case ftEscdecimal:
			if isDigit(c&0xff) == false {
				return nil, nil, errors.New("mixture of escaped digit and non-digit")
			}
			value = value * 10
			value = value + digitvalue[c]
			digits++
			if digits == 3 {
				if value > 255 {
					return nil, nil, errors.New("escaped decimal is too large")
				}
				count++
				if count > MAX_LABEL_LEN {
					return nil, nil, errors.New("lable is too long")
				}
				if downcase {
					data = append(data, maptolower[value])
				} else {
					data = append(data, byte(value))
				}
				state = ftOrdinary
			}
		default:
			panic("impossible state")
		}
	}

	if done == false {
		if len(data) == MAX_WIRE {
			return nil, nil, errors.New("too long name")
		}
		if start != end {
			panic("start should equal to end")
		}
		if state != ftOrdinary {
			return nil, nil, errors.New("incomplete textural name")
		} else {
			if count == 0 {
				panic("count shouldn't equal to zero")
			}
			data[offsets[len(offsets)-1]] = byte(count)
			offsets = append(offsets, byte(len(data)))
			data = append(data, 0)
		}
	}
	return data, offsets, nil
}

var Root = &Name{[]byte{0}, []byte{0}, 1, 1}

func NewName(name string, downcase bool) (*Name, error) {
	raw, offsets, err := stringParse([]byte(name), 0, uint(len(name)), downcase)
	if err != nil {
		return nil, err
	} else {
		return &Name{raw, offsets, uint(len(raw)), uint(len(offsets))}, nil
	}
}

const (
	fwStart uint = iota
	fwOrdinary
	fwNewCurrent
)

func NameFromString(s string) (*Name, error) {
	return NewName(s, true)
}

func NameFromStringUnsafe(s string) *Name {
	name, err := NewName(s, true)
	if err != nil {
		panic("unvalid name" + s)
	}
	return name
}

func NameFromWire(buf *util.InputBuffer, downcase bool) (*Name, error) {
	n := uint(0)
	nused := uint(0)
	done := false
	//5, 15 is the experienced value for label and name len
	offsets := make([]byte, 0, 5)
	raw := make([]byte, 0, 15)
	seenPointer := false
	state := fwStart
	cused := uint(0)
	current := buf.Position()
	posBegin := current
	biggestPointer := current
	newCurrent := uint(0)

	for current < buf.Len() && done == false {
		c, _ := buf.ReadUint8()
		current += 1

		if seenPointer == false {
			cused++
		}

		switch state {
		case fwStart:
			if c <= MAX_LABEL_LEN {
				offsets = append(offsets, byte(nused))
				if nused+uint(c)+1 > MAX_WIRE {
					return nil, errors.New("too long name")
				}

				nused = nused + uint(c) + 1
				raw = append(raw, c)
				if c == 0 {
					done = true
				}
				n = uint(c)
				state = fwOrdinary
			} else if c&COMPRESS_POINTER_MARK8 == COMPRESS_POINTER_MARK8 {
				newCurrent = uint(c & ^uint8(COMPRESS_POINTER_MARK8))
				n = 1
				state = fwNewCurrent
			} else {
				return nil, errors.New("unknown label character")
			}
		case fwOrdinary:
			if downcase {
				c = maptolower[c]
			}
			raw = append(raw, c)
			n--
			if n == 0 {
				state = fwStart
			}
		case fwNewCurrent:
			newCurrent *= 256
			newCurrent += uint(c)
			n--
			if n != 0 {
				break
			}
			if newCurrent >= biggestPointer {
				return nil, errors.New("bad compression pointer")
			}
			biggestPointer = newCurrent
			current = newCurrent
			buf.SetPosition(current)
			seenPointer = true
			state = fwStart
		default:
			panic("impossible state")
		}
	}

	if done == false {
		return nil, errors.New("imcomplete wire format")
	}

	buf.SetPosition(posBegin + cused)
	return &Name{raw, offsets, uint(len(raw)), uint(len(offsets))}, nil
}

func (name *Name) Length() uint {
	return name.length
}

func (name *Name) LabelCount() uint {
	return name.labelCount
}

func (name *Name) String(omitFinalDot bool) string {
	var result bytes.Buffer
	for i := uint(0); i < name.length; {
		count := int(name.raw[i])
		i++

		if count == 0 {
			if !omitFinalDot || result.Len() == 0 {
				result.WriteRune('.')
			}
			break
		}

		if count > MAX_LABEL_LEN {
			panic("too long label")
		}
		if result.Len() != 0 {
			result.WriteRune('.')
		}

		for count > 0 {
			count--
			c := rune(name.raw[i])
			i++
			switch c {
			case 0x22, 0x28, 0x29, 0x2E, 0x3B, 0x5C, 0x40, 0x24: //" ( ) . ; \\ @ $
				result.WriteRune('\\')
				result.WriteRune(c)
			default:
				if c > 0x20 && c < 0x7f {
					result.WriteRune(c)
				} else {
					result.WriteRune(0x5c)
					result.WriteRune(0x30 + ((c / 100) % 10))
					result.WriteRune(0x30 + ((c / 10) % 10))
					result.WriteRune(0x30 + (c % 10))
				}
			}
		}
	}
	return result.String()
}

func min(n1 uint, n2 uint) uint {
	if n1 > n2 {
		return n2
	} else {
		return n1
	}
}

func (n1 *Name) Compare(n2 *Name, caseSensitive bool) NameComparisonResult {
	l1 := n1.labelCount
	l2 := n2.labelCount
	ldiff := int(l1) - int(l2)
	minl := min(l1, l2)
	nlabels := 0
	for minl > 0 {
		minl--
		l1--
		l2--
		ps1 := n1.offsets[l1]
		ps2 := n2.offsets[l2]
		c1 := n1.raw[ps1]
		c2 := n2.raw[ps2]
		ps1++
		ps2++

		cdiff := int(c1) - int(c2)
		mincount := min(uint(c1), uint(c2))

		for mincount > 0 {
			label1 := n1.raw[ps1]
			label2 := n2.raw[ps2]
			var chdiff int
			if caseSensitive {
				chdiff = int(label1) - int(label2)
			} else {
				chdiff = int(maptolower[label1]) - int(maptolower[label2])
			}

			if chdiff != 0 {
				return NameComparisonResult{chdiff, nlabels, COMMONANCESTOR}
			}
			mincount--
			ps1++
			ps2++
		}

		if cdiff != 0 {
			if nlabels == 0 {
				return NameComparisonResult{cdiff, nlabels, NONE}
			} else {
				return NameComparisonResult{cdiff, nlabels, COMMONANCESTOR}
			}
		}
		nlabels++
	}

	if ldiff < 0 {
		return NameComparisonResult{ldiff, nlabels, SUPERDOMAIN}
	} else if ldiff > 0 {
		return NameComparisonResult{ldiff, nlabels, SUBDOMAIN}
	} else {
		return NameComparisonResult{ldiff, nlabels, EQUAL}
	}
}

func (n1 *Name) CaseSensitiveEquals(n2 *Name) bool {
	if n1.length != n2.length || n1.labelCount != n2.labelCount {
		return false
	}
	return bytes.Compare(n1.raw, n2.raw) == 0
}

func (n1 *Name) Equals(n2 *Name) bool {
	if n1.length != n2.length || n1.labelCount != n2.labelCount {
		return false
	}

	pos := 0
	for l := n1.labelCount; l > 0; l-- {
		count := n1.raw[pos]
		if count != n2.raw[pos] {
			return false
		}
		pos++

		for count > 0 {
			if maptolower[n1.raw[pos]] != maptolower[n2.raw[pos]] {
				return false
			}
			count--
			pos++
		}
	}

	return true
}

func (name *Name) IsWildCard() bool {
	return name.length >= 2 && name.raw[0] == 1 && name.raw[1] == '*'
}

func (name *Name) Concat(suffixes ...*Name) (*Name, error) {
	if len(suffixes) == 1 && suffixes[0].IsRoot() {
		return name, nil
	}

	finalLength := name.length
	finalLabelCount := name.labelCount
	suffixCount := uint(len(suffixes))
	for _, suffix := range suffixes {
		finalLength += suffix.length - 1
		finalLabelCount += suffix.labelCount - 1
	}

	if finalLength > MAX_WIRE {
		return nil, errors.New("names are too long to concat")
	} else if finalLabelCount > MAX_LABELS {
		return nil, errors.New("names has too many labels to concat")
	}

	raw := make([]byte, finalLength)
	copy(raw, name.raw[0:name.length-1])
	copyedLen := name.length - 1
	for _, suffix := range suffixes[:suffixCount-1] {
		copy(raw[copyedLen:], suffix.raw[0:suffix.length-1])
		copyedLen += suffix.length - 1
	}
	copy(raw[copyedLen:], suffixes[suffixCount-1].raw)

	offsets := make([]byte, finalLabelCount)
	copy(offsets, name.offsets)
	copyedLen = name.labelCount
	for _, suffix := range suffixes {
		lastOffset := offsets[copyedLen-1]
		copy(offsets[copyedLen:], suffix.offsets[1:suffix.labelCount])
		for i := copyedLen; i < copyedLen+suffix.labelCount-1; i++ {
			offsets[i] += byte(lastOffset)
		}
		copyedLen += suffix.labelCount - 1
	}
	return &Name{raw, offsets, uint(len(raw)), uint(len(offsets))}, nil
}

//"a.b.c" Subtrace "b.c" == "a"
//caller should make sure name endWith suffix
func (name *Name) Subtract(suffix *Name) (*Name, error) {
	return name.Split(0, name.LabelCount()-suffix.LabelCount())
}

func (name *Name) Reverse() *Name {
	if name.labelCount == 1 {
		return Root
	}

	raw := make([]byte, name.length-1, name.length)
	offsets := make([]byte, 0, name.labelCount)
	labelLen := byte(0)
	for i := int(name.labelCount - 2); i >= 0; i-- {
		labelStart := name.offsets[i]
		labelEnd := name.offsets[i+1]
		copy(raw[labelLen:], name.raw[labelStart:labelEnd])
		offsets = append(offsets, labelLen)
		labelLen += labelEnd - labelStart
	}
	raw = append(raw, 0)
	offsets = append(offsets, labelLen)
	return &Name{raw, offsets, name.length, name.labelCount}
}

func (name *Name) Split(startLabel uint, labelCount uint) (*Name, error) {
	if labelCount == 0 || labelCount > name.labelCount || startLabel+labelCount > name.labelCount {
		return nil, errors.New("split range isn't valid")
	}

	if startLabel+labelCount == name.labelCount {
		if startLabel == 0 {
			return name, nil
		} else {
			offsets := make([]byte, labelCount)
			firstOffset := name.offsets[startLabel]
			copy(offsets, name.offsets[startLabel:])
			raw := name.raw[offsets[0]:name.length]
			for i := uint(0); i < labelCount; i++ {
				offsets[i] -= firstOffset
			}
			return &Name{raw, offsets, uint(len(raw)), labelCount}, nil
		}
	} else {
		offsets := make([]byte, labelCount+1)
		firstOffset := name.offsets[startLabel]
		copy(offsets, name.offsets[startLabel:startLabel+labelCount+1])
		raw := make([]byte, offsets[labelCount]-offsets[0]+1)
		copy(raw, name.raw[offsets[0]:offsets[labelCount]])
		for i := uint(0); i < labelCount+1; i++ {
			offsets[i] -= firstOffset
		}
		raw[offsets[labelCount]] = 0
		return &Name{raw, offsets, uint(len(raw)), labelCount + 1}, nil
	}
}

func (name *Name) Parent(level uint) (*Name, error) {
	return name.Split(level, name.labelCount-level)
}

//this is the only exception which will modify the name itself
//since it only change the content of the raw data, the impact
//on other shared name is limited
func (name *Name) Downcase() {
	lc := name.labelCount
	p := 0

	for lc > 0 {
		lc--

		ll := name.raw[p]
		p++
		for ll > 0 {
			name.raw[p] = maptolower[name.raw[p]]
			p++
			ll--
		}
	}
}

func (name *Name) StripLeft(c uint) (*Name, error) {
	if c >= name.labelCount {
		return nil, errors.New("strip too many labels")
	}

	if c == 0 {
		return name, nil
	}

	startPos := name.offsets[c]
	newLabelCount := name.labelCount - c
	offsets := make([]byte, newLabelCount)
	copy(offsets, name.offsets[c:])
	for i := uint(0); i < newLabelCount; i++ {
		offsets[i] -= startPos
	}

	return &Name{name.raw[startPos:], offsets, name.length - uint(startPos), uint(len(offsets))}, nil
}

func (name *Name) StripRight(c uint) (*Name, error) {
	if c >= name.labelCount {
		return nil, errors.New("strip too many labels")
	}

	if c == 0 {
		return name, nil
	}

	labelIndex := name.labelCount - c - 1
	endPos := name.offsets[labelIndex]
	raw := make([]byte, endPos+1)
	copy(raw, name.raw[0:endPos])
	raw[endPos] = 0
	offsets := name.offsets[0 : labelIndex+1]
	return &Name{raw, offsets, uint(endPos + 1), name.labelCount - c}, nil
}

func (name *Name) Hash(caseSensitive bool) uint32 {
	return hashRaw(name.raw, caseSensitive)
}

func hashRaw(raw []byte, caseSensitive bool) uint32 {
	hashLen := len(raw)
	hash := uint32(0)
	if caseSensitive {
		for i := 0; i < hashLen; i++ {
			hash ^= uint32(raw[i]) + 0x9e3779b9 + (hash << 6) + (hash >> 2)
		}
	} else {
		for i := 0; i < hashLen; i++ {
			hash ^= uint32(maptolower[raw[i]]) + 0x9e3779b9 + (hash << 6) + (hash >> 2)
		}
	}
	return hash
}

func (name *Name) Rend(render *MsgRender) {
	render.WriteName(name, true)
}

func (name *Name) ToWire(buf *util.OutputBuffer) {
	buf.WriteData(name.raw)
}

func (name *Name) IsRoot() bool {
	return name.LabelCount() == 1
}

func (name *Name) IsSubDomain(parent *Name) bool {
	if name.length < parent.length || name.labelCount < parent.labelCount {
		return false
	}

	i := name.length - 1
	for j := parent.length - 1; j > 0; j -= 1 {
		if maptolower[parent.raw[j]] != maptolower[name.raw[i]] {
			return false
		}
		i -= 1
	}
	return true
}
