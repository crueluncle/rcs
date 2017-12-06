/*gob序列化方法实际就是一种协议封包解包方法(gob传输不会出现粘包问题)，但gob在解包时必须已知封包时的具体结构，要在一条连接上传输多种数据结构(发送顺序未知)必须要有另外的方式告知对方本次发送的数据结构信息
因此可将要发送的对象组织成type信息+reflect.value信息的形式来发送(两次发送原子化)，接收方读取两次(两次读取原子化)依据type信息来动态构建对象(简单DI库,接收方需实现注册需要接收的消息类型)
1.use the  reflect package to decode  any type struct data to interface{} with gob
2.must regitst the struct in advance,otherwise the recvier connot decode the struct
*/

package utils

import (
	"bufio"
	"encoding/gob"
	"errors"
	"io"
	//	"crypto/aes"
	"reflect"
	"sync"
)

var msgTypes map[string]reflect.Type

type Codecer interface {
	Write(interface{}) error
	Read(chan<- interface{}) error
	Close() error
}
type msg struct { //任何消息组织成此结构：类型字符串+反射值
	msgtyp  string
	msgdata reflect.Value
}
type gobCodecer struct {
	rwc    io.ReadWriteCloser
	wlock  *sync.Mutex  // 确保发送操作原子化
	enc    *gob.Encoder //使用gob编码
	dec    *gob.Decoder //使用gob解码
	encBuf *bufio.Writer
	closed bool
}

func NewCodecer(io io.ReadWriteCloser) Codecer {
	encBuf := bufio.NewWriter(io)
	return &gobCodecer{
		rwc:    io,
		wlock:  new(sync.Mutex),
		enc:    gob.NewEncoder(encBuf),
		dec:    gob.NewDecoder(io),
		encBuf: encBuf,
		closed: false,
	}
}

func (gc *gobCodecer) Write(msgs interface{}) error {
	if gc.closed == true || gc.dec == nil || gc.enc == nil {
		return errors.New("gobcodecer is invalid!")
	}
	gc.wlock.Lock()
	defer gc.wlock.Unlock()
	var pack msg
	pack.msgtyp = reflect.TypeOf(msgs).String()
	pack.msgdata = reflect.ValueOf(msgs)
	if err := gc.enc.Encode(pack.msgtyp); err != nil {
		return err
	}
	if err := gc.enc.EncodeValue(pack.msgdata); err != nil {
		return err
	}
	return gc.encBuf.Flush()
}
func (gc *gobCodecer) Read(rcvch chan<- interface{}) error { //持续读,业务层只需从rcvch中提取消息进行处理[一条tcp连接上只能单goroutine执行,多goroutine执行读无意义]
	var tp = new(string)
	var val reflect.Value
	for {
		if gc.closed == true || gc.dec == nil || gc.enc == nil {
			return errors.New("gobcodecer is invalid!")
		}
		if err := gc.dec.Decode(tp); err != nil {
			return err
		}
		typ, ok := msgTypes[*tp]
		if !ok {
			if err := gc.dec.DecodeValue(reflect.ValueOf(nil)); err != nil {
				return err
			}
			return errors.New(*tp + ":type not regiested.")
		}
		val = reflect.New(typ)
		if err := gc.dec.DecodeValue(val); err != nil {
			return err
		}
		rcvch <- val.Elem().Interface()
	}
}
func (gc *gobCodecer) Close() error {
	if err := gc.rwc.Close(); err != nil {
		return err
	}
	gc.closed = true
	gc.enc = nil
	gc.dec = nil
	return nil
}
func MsgTypeRegist(msg interface{}) {
	v := reflect.ValueOf(msg)
	typeRegist(v)
}
func typeRegist(val reflect.Value) {
	msgTypes[val.Type().String()] = val.Type()
}

func init() {
	msgTypes = make(map[string]reflect.Type)
	MsgTypeRegist(int(0))
	MsgTypeRegist(int8(0))
	MsgTypeRegist(int16(0))
	MsgTypeRegist(int32(0))
	MsgTypeRegist(int64(0))
	MsgTypeRegist(uint(0))
	MsgTypeRegist(uint8(0))
	MsgTypeRegist(uint16(0))
	MsgTypeRegist(uint32(0))
	MsgTypeRegist(uint64(0))
	MsgTypeRegist(float32(0))
	MsgTypeRegist(float64(0))
	MsgTypeRegist(complex64(0i))
	MsgTypeRegist(complex128(0i))
	MsgTypeRegist(uintptr(0))
	MsgTypeRegist(false)
	MsgTypeRegist("")
	MsgTypeRegist([]byte(nil))
	MsgTypeRegist([]int(nil))
	MsgTypeRegist([]int8(nil))
	MsgTypeRegist([]int16(nil))
	MsgTypeRegist([]int32(nil))
	MsgTypeRegist([]int64(nil))
	MsgTypeRegist([]uint(nil))
	MsgTypeRegist([]uint8(nil))
	MsgTypeRegist([]uint16(nil))
	MsgTypeRegist([]uint32(nil))
	MsgTypeRegist([]uint64(nil))
	MsgTypeRegist([]float32(nil))
	MsgTypeRegist([]float64(nil))
	MsgTypeRegist([]complex64(nil))
	MsgTypeRegist([]complex128(nil))
	MsgTypeRegist([]uintptr(nil))
	MsgTypeRegist([]bool(nil))
	MsgTypeRegist([]string(nil))
}
