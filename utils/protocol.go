package utils

import (
	"bytes"
	"encoding/binary"
)

type ProtocolAnalyzer interface {
	Enpack([]byte) []byte
	Depack([]byte, chan<- []byte)
	GetlagecyMsg() []byte
}

func NewProtocolAnalyzer() ProtocolAnalyzer {
	return DefaultProtocal()
}
func DefaultProtocal() ProtocolAnalyzer {
	return &myProtocal{
		headerflag:   []byte(`\^\$`),
		constmlength: 4,
		lagecyMsg:    make([]byte, 0),
	}
}

type myProtocal struct {
	headerflag   []byte
	constmlength int
	lagecyMsg    []byte
}

func (dp *myProtocal) Enpack(message []byte) []byte {
	return append(append(dp.headerflag, IntToBytes(len(message))...), message...)
}

func (dp *myProtocal) Depack(srcdata []byte, pkgChan chan<- []byte) {
	length := len(srcdata)
	constheaderlength := len(dp.headerflag)
	constheader := dp.headerflag
	constmlength := dp.constmlength
	var i int
	for i = 0; i < length; i++ {
		if length < i+constheaderlength+constmlength {
			break
		}
		hindex := bytes.Index(srcdata[i:], constheader) + i
		if hindex >= i {
			messageLength := BytesToInt(srcdata[hindex+constheaderlength : hindex+constheaderlength+constmlength])
			if length < hindex+constheaderlength+constmlength+messageLength {
				break
			}
			data := srcdata[hindex+constheaderlength+constmlength : hindex+constheaderlength+constmlength+messageLength]
			pkgChan <- data
			i = hindex + constheaderlength + constmlength + messageLength - 1
		}
	}
	if i == length {
		dp.lagecyMsg = make([]byte, 0)
	}
	dp.lagecyMsg = srcdata[i:]
}

func (dp *myProtocal) GetlagecyMsg() []byte {
	return dp.lagecyMsg
}

func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}
