package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Header struct {
	Magic uint32
	Len   uint32
}

const MagicNumber = 0x12345678 // Header.Magic uint32, 即 4 个字节, 8 位十六进制数

var HeaderSize = binary.Size(Header{})

// Pack 将 msg 封装成 Header + body
func Pack(msg []byte) ([]byte, error) {
	header := Header{Magic: MagicNumber, Len: uint32(len(msg))}

	buf := new(bytes.Buffer)
	buf.Grow(HeaderSize + len(msg))

	if err := binary.Write(buf, binary.BigEndian, header); err != nil {
		return nil, fmt.Errorf("write header to buffer err: %w", err)
	}
	if _, err := buf.Write(msg); err != nil {
		return nil, fmt.Errorf("write body to buffer err: %w", err)
	}
	return buf.Bytes(), nil
}

// Read 从 reader 中读取数据并解析出 Header 和 body
func Read(reader io.Reader) (Header, []byte, error) {
	var header Header
	headerRaw := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, headerRaw); err != nil {
		return header, nil, fmt.Errorf("illegal header: %w", err)
	}
	if err := binary.Read(bytes.NewBuffer(headerRaw), binary.BigEndian, &header); err != nil {
		return header, nil, fmt.Errorf("read header to struct err: %w", err)
	}
	if header.Magic != MagicNumber {
		return header, nil, fmt.Errorf("illegal magic number: %d", header.Magic)
	}

	body := make([]byte, header.Len)
	if _, err := io.ReadFull(reader, body); err != nil {
		return header, nil, fmt.Errorf("read body err: %w", err)
	}
	return header, body, nil
}
