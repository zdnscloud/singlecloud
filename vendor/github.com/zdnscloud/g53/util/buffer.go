package util

import (
	"errors"
)

type InputBuffer struct {
	pos     uint
	data    []byte
	datalen uint
}

func NewInputBuffer(bytes []byte) *InputBuffer {
	buf := &InputBuffer{}
	buf.SetData(bytes)
	return buf
}

func (buf *InputBuffer) SetData(bytes []byte) {
	buf.pos = 0
	buf.data = bytes
	buf.datalen = uint(len(bytes))
}

func (buf *InputBuffer) Len() uint {
	return buf.datalen
}

func (buf *InputBuffer) Position() uint {
	return buf.pos
}

func (buf *InputBuffer) SetPosition(p uint) error {
	if p > buf.datalen {
		return errors.New("buffer overflow")
	}
	buf.pos = p
	return nil
}

func (buf *InputBuffer) ReadUint8() (uint8, error) {
	if buf.pos+1 > buf.datalen {
		return 0, errors.New("read beyond end of buffer")
	}

	b := buf.data[buf.pos]
	buf.pos += 1
	return uint8(b), nil
}

func (buf *InputBuffer) ReadUint16() (uint16, error) {
	if buf.pos+2 > buf.datalen {
		return 0, errors.New("read beyond end of buffer")
	}

	p := buf.pos
	short := uint16(buf.data[p]) << 8
	short |= uint16(buf.data[p+1])
	buf.pos += 2
	return uint16(short), nil
}

func (buf *InputBuffer) ReadUint32() (uint32, error) {
	if buf.pos+4 > buf.datalen {
		return 0, errors.New("read beyond end of buffer")
	}

	p := buf.pos
	i := uint32(buf.data[p]) << 24
	i |= uint32(buf.data[p+1]) << 16
	i |= uint32(buf.data[p+2]) << 8
	i |= uint32(buf.data[p+3])
	buf.pos += 4
	return i, nil
}

func (buf *InputBuffer) ReadBytes(length uint) ([]byte, error) {
	if buf.pos+length > buf.datalen {
		return nil, errors.New("read beyond end of buffer")
	}

	p := buf.pos
	data := buf.data[p : p+length]
	buf.pos += length
	return data, nil
}

type OutputBuffer struct {
	data []uint8
}

func NewOutputBuffer(length uint) *OutputBuffer {
	return &OutputBuffer{
		data: make([]uint8, 0, length),
	}
}

func (out *OutputBuffer) Len() uint {
	return uint(len(out.data))
}

func (out *OutputBuffer) Capacity() uint {
	return uint(cap(out.data))
}

func (out *OutputBuffer) Data() []uint8 {
	return out.data
}

func (out *OutputBuffer) At(pos uint) (uint8, error) {
	if pos < out.Len() {
		return out.data[pos], nil
	} else {
		return 0, errors.New("out of range")
	}
}

func (out *OutputBuffer) Skip(length uint) {
	l := out.Len() + length
	out.ensureSpace(l)
	for cl := out.Len(); cl < l; cl++ {
		out.data = append(out.data, 0)
	}
}

func (out *OutputBuffer) Trim(length uint) error {
	if length > out.Len() {
		return errors.New("trim too large from output buffer")
	} else {
		out.data = out.data[0:(out.Len() - length)]
		return nil
	}
}

func (out *OutputBuffer) ensureSpace(length uint) {
	c := out.Capacity()
	if c < length {
		if c == 0 {
			c = 1024
		}
		for c < length {
			c = c * 2
		}
		d := make([]uint8, length, c)
		copy(d, out.data)
		out.data = d
	}
}

func (out *OutputBuffer) Clear() {
	out.data = out.data[:0]
}

func (out *OutputBuffer) WriteUint8(data uint8) {
	out.data = append(out.data, data)
}

func (out *OutputBuffer) WriteUint8At(data uint8, pos uint) error {
	if pos+1 > out.Len() {
		return errors.New("write at invalid pos")
	} else {
		out.data[pos] = data
		return nil
	}
}

func (out *OutputBuffer) WriteUint16(data uint16) {
	out.data = append(out.data, uint8((data&0xff00)>>8), uint8(data&0x00ff))
}

func (out *OutputBuffer) WriteUint16At(data uint16, pos uint) error {
	if pos+2 > out.Len() {
		return errors.New("write at invalid pos")
	} else {
		out.data[pos] = uint8((data & 0xff00) >> 8)
		out.data[pos+1] = uint8(data & 0x00ff)
		return nil
	}
}

func (out *OutputBuffer) WriteUint32(data uint32) {
	out.data = append(out.data,
		uint8((data&0xff000000)>>24),
		uint8((data&0x00ff0000)>>16),
		uint8((data&0x0000ff00)>>8),
		uint8(data&0x000000ff))
}

func (out *OutputBuffer) WriteData(data []uint8) {
	out.data = append(out.data, data...)
}
