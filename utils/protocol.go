//通讯协议处理,定义了一种通用性文本协议
package utils

import (
	"bytes"
	"encoding/binary"
)

/*
头部标记--消息体长度--消息体(业务消息内容)
*/
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
		constmlength: 4, //默认最大支持int32字节长度的消息
		lagecyMsg:    make([]byte, 0),
	}
}

type myProtocal struct {
	headerflag   []byte
	constmlength int    //消息体长度字段的长度(int32)
	lagecyMsg    []byte //仅解包会用到
}

//封包
func (dp *myProtocal) Enpack(message []byte) []byte {
	return append(append(dp.headerflag, IntToBytes(len(message))...), message...)
}

//解包
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

//获取截断消息
func (dp *myProtocal) GetlagecyMsg() []byte {
	return dp.lagecyMsg
}

//整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

//字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}
