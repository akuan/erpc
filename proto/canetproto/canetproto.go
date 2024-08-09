// Package canetProto is implemented canet style socket communication protocol.
//
// Copyright 2024 akuan. All Rights Reserved.
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
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/erpc/v7/codec"
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
		id:   'c',
		name: "canet",
		r:    rw,
		w:    rw,
	}
}

// Version returns the protocol's id and name.
func (r *canetProto) Version() (byte, string) {
	return r.id, r.name
}

// Pack writes the Message into the connection. path=tid,data=body
// NOTE: Make sure to write only once or there will be package contamination!
func (r *canetProto) Pack(m socket.Message) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	//A canet data frame can only send 8 bytes of data
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}
	tid, err := strconv.Atoi(m.ServiceMethod())
	if err != nil {
		return err
	}
	Max_frame_size := 8
	dataLen := len(bodyBytes)
	frameDataLen := dataLen
	remainLen := dataLen
	for remainLen > 0 {
		//over than 8 bytes, split
		if remainLen > Max_frame_size {
			frameDataLen = Max_frame_size
		} else {
			frameDataLen = remainLen
		}
		//canet frame header
		bb.WriteByte(byte(frameDataLen))
		//tid
		err = binary.Write(bb, binary.BigEndian, uint32(tid))
		if err != nil {
			return err
		}
		//write data
		writedLen := dataLen - remainLen
		bb.Write(bodyBytes[writedLen:frameDataLen])
		//feed to 8 bytes
		if frameDataLen < Max_frame_size {
			for i := 0; i < Max_frame_size-frameDataLen; i++ {
				bb.WriteByte(0)
			}
		}
		remainLen -= frameDataLen
		err = m.SetSize(uint32(bb.Len()))
		if err != nil {
			return err
		}
		_, err = r.w.Write(bb.B)
		if err != nil {
			return err
		}
		bb.Reset()
	}
	return err
}

// Unpack reads bytes from the connection to the Message.
// NOTE: Concurrent unsafe!
func (r *canetProto) Unpack(m socket.Message) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)
	r.rMu.Lock()
	defer r.rMu.Unlock()
	// size
	bb.ChangeLen(13)
	_, err := io.ReadFull(r.r, bb.B)
	if err != nil {
		return err
	}
	m.SetMtype(erpc.TypePush)
	fmt.Println(bb.B)
	payload := bb.B[5:]
	m.SetServiceMethod("/canet")
	m.SetBodyCodec(codec.ID_CANET)
	m.SetSize(uint32(len(payload)))
	return m.UnmarshalBody(payload)
	// // read message
	// err := r.readMessage(bb, m)
	// if err != nil {
	// 	return err
	// }
	// // do transfer pipe
	// data, err := m.XferPipe().OnUnpack(bb.B)
	// if err != nil {
	// 	return err
	// }
	// // header
	// data, err = r.readHeader(data, m)
	// if err != nil {
	// 	return err
	// }
	// body
	//return r.readBody(data, m)
	//return nil
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

	m.SetMtype(erpc.TypePush)
	// type
	//m.SetMtype(data[0])
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
