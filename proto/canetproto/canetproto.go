// Package canetProto is implemented canet style socket communication protocol.
//
// Copyright 2018 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package canetproto

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strconv"
	"sync"

	"github.com/andeya/erpc/v7/socket"
	"github.com/andeya/erpc/v7/utils"
	"github.com/andeya/goutil"
)

var ()

// canetProto fast socket communication protocol.
type canetProto struct {
	r    io.Reader
	w    io.Writer
	rMu  sync.Mutex
	name string
	id   byte
}

// CanetProtoFunc is creation function of fast socket protocol.
// NOTE: it is the use for canet data send and receive, actually it doesn't have header data.
var CanetProtoFunc = func(rw socket.IOWithReadBuffer) socket.Proto {
	return &canetProto{
		id:   6,
		name: "raw",
		r:    rw,
		w:    rw,
	}
}

// Version returns the protocol's id and name.
func (r *canetProto) Version() (byte, string) {
	return r.id, r.name
}

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
// nolint:ineffassign
func (r *canetProto) Pack(m socket.Message) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	// fake size
	err := binary.Write(bb, binary.BigEndian, uint32(0))

	// transfer pipe
	bb.WriteByte(byte(m.XferPipe().Len()))
	bb.Write(m.XferPipe().IDs())

	prefixLen := bb.Len()

	// header
	err = r.writeHeader(bb, m)
	if err != nil {
		return err
	}

	// body
	err = r.writeBody(bb, m)
	if err != nil {
		return err
	}

	// do transfer pipe
	payload, err := m.XferPipe().OnPack(bb.B[prefixLen:])
	if err != nil {
		return err
	}
	bb.B = append(bb.B[:prefixLen], payload...)

	// set and check message size
	err = m.SetSize(uint32(bb.Len()))
	if err != nil {
		return err
	}

	// reset real size
	binary.BigEndian.PutUint32(bb.B, m.Size())

	// real write
	_, err = r.w.Write(bb.B)
	if err != nil {
		return err
	}

	return err
}

func (r *canetProto) writeHeader(bb *utils.ByteBuffer, m socket.Message) error {
	seqStr := strconv.FormatInt(int64(m.Seq()), 36)
	bb.WriteByte(byte(len(seqStr)))
	bb.Write(goutil.StringToBytes(seqStr))

	bb.WriteByte(m.Mtype())

	serviceMethod := goutil.StringToBytes(m.ServiceMethod())
	serviceMethodLength := len(serviceMethod)
	if serviceMethodLength > math.MaxUint8 {
		return errors.New("raw proto: not support service method longer than 255")
	}
	bb.WriteByte(byte(serviceMethodLength))
	bb.Write(serviceMethod)
	statusBytes := m.Status(true).EncodeQuery()
	binary.Write(bb, binary.BigEndian, uint16(len(statusBytes)))
	bb.Write(statusBytes)

	metaBytes := m.Meta().QueryString()
	binary.Write(bb, binary.BigEndian, uint16(len(metaBytes)))
	bb.Write(metaBytes)
	return nil
}

func (r *canetProto) writeBody(bb *utils.ByteBuffer, m socket.Message) error {
	bb.WriteByte(m.BodyCodec())
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}
	bb.Write(bodyBytes)
	return nil
}

// Unpack reads bytes from the connection to the Message.
// NOTE: Concurrent unsafe!
func (r *canetProto) Unpack(m socket.Message) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	// read message
	err := r.readMessage(bb, m)
	if err != nil {
		return err
	}
	// do transfer pipe
	data, err := m.XferPipe().OnUnpack(bb.B)
	if err != nil {
		return err
	}
	// header
	data, err = r.readHeader(data, m)
	if err != nil {
		return err
	}
	// body
	return r.readBody(data, m)
}

func (r *canetProto) readMessage(bb *utils.ByteBuffer, m socket.Message) error {
	r.rMu.Lock()
	defer r.rMu.Unlock()

	// size
	bb.ChangeLen(4)
	_, err := io.ReadFull(r.r, bb.B)
	if err != nil {
		return err
	}
	_lastSize := binary.BigEndian.Uint32(bb.B)
	if err = m.SetSize(_lastSize); err != nil {
		return err
	}
	lastSize := int(_lastSize)
	lastSize, err = minus(lastSize, 4)
	if err != nil {
		return err
	}
	bb.ChangeLen(lastSize)

	// transfer pipe
	_, err = io.ReadFull(r.r, bb.B[:1])
	if err != nil {
		return err
	}
	var xferLen = bb.B[0]
	if xferLen > 0 {
		_, err = io.ReadFull(r.r, bb.B[:xferLen])
		if err != nil {
			return err
		}
		err = m.XferPipe().Append(bb.B[:xferLen]...)
		if err != nil {
			return err
		}
	}
	lastSize, err = minus(lastSize, 1+int(xferLen))
	if err != nil {
		return err
	}
	// read last all
	bb.ChangeLen(lastSize)
	_, err = io.ReadFull(r.r, bb.B)
	return err
}

func minus(a int, b int) (int, error) {
	r := a - b
	if r < 0 || b < 0 {
		return a, errors.New("raw proto: bad package")
	}
	return r, nil
}

func (r *canetProto) readHeader(data []byte, m socket.Message) ([]byte, error) {
	// seq
	seqLen := data[0]
	data = data[1:]
	seq, err := strconv.ParseInt(goutil.BytesToString(data[:seqLen]), 36, 32)
	if err != nil {
		return nil, err
	}
	m.SetSeq(int32(seq))
	data = data[seqLen:]

	// type
	m.SetMtype(data[0])
	data = data[1:]

	// service method
	serviceMethodLen := data[0]
	data = data[1:]
	m.SetServiceMethod(string(data[:serviceMethodLen]))
	data = data[serviceMethodLen:]

	// status
	statusLen := binary.BigEndian.Uint16(data)
	data = data[2:]
	m.Status(true).DecodeQuery(data[:statusLen])
	data = data[statusLen:]

	// meta
	metaLen := binary.BigEndian.Uint16(data)
	data = data[2:]
	m.Meta().ParseBytes(data[:metaLen])
	data = data[metaLen:]

	return data, nil
}

func (r *canetProto) readBody(data []byte, m socket.Message) error {
	m.SetBodyCodec(data[0])
	return m.UnmarshalBody(data[1:])
}
