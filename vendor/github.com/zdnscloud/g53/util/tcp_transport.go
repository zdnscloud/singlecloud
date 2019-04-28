package util

import (
	"encoding/binary"
	"io"
	"net"
	"time"
)

const tcpTimeout = 5 * time.Second

func NewTCPConn(address string) (*net.TCPConn, error) {
	conn, err := net.DialTimeout("tcp", address, tcpTimeout)
	if err != nil {
		return nil, err
	}

	return conn.(*net.TCPConn), nil
}

func TCPWrite(data []byte, conn *net.TCPConn) error {
	size := uint16(len(data))
	if err := binary.Write(conn, binary.BigEndian, &size); err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(tcpTimeout))
	_, err := conn.Write(data)
	return err
}

func TCPRead(conn *net.TCPConn) ([]byte, error) {
	var msgSize uint16
	conn.SetReadDeadline(time.Now().Add(tcpTimeout))
	if err := binary.Read(conn, binary.BigEndian, &msgSize); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(tcpTimeout))
	buf := make([]byte, msgSize)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}

	return buf, nil
}
