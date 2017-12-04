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
type msg struct {
	msgtyp  string
	msgdata reflect.Value
}
type gobCodecer struct {
	rwc    io.ReadWriteCloser
	wlock  *sync.Mutex
	enc    *gob.Encoder
	dec    *gob.Decoder
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
func (gc *gobCodecer) Read(rcvch chan<- interface{}) error {
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
