package main

import (
	"fmt"
	"log"
	"time"

	"github.com/akuan/erpc/v7/proto/canetproto"
	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_client" $GOFILE

func main() {
	if goutil.IsGoTest() {
		//t.Log("skip test in go test")
		return
	}
	defer erpc.SetLoggerLevel("DEBUG")()

	cli := erpc.NewPeer(erpc.PeerConfig{
		Network: "tcp4", RedialTimes: -1, RedialInterval: time.Millisecond * 20, DialTimeout: time.Second * 5,
		DefaultBodyCodec: "canet",
	})
	defer cli.Close()
	//cli.SetTLSConfig(erpc.GenerateTLSConfigForClient())

	//cli.RoutePush(new(Push))
	//cli.RoutePush(new((*Canet).Canet))
	cli.RoutePushFunc((*Canet).Canet)
	//cli.SubRoute("/canet").RoutePush(CanetProcess)
	//cli.SubRoute("/cli").
	//	RoutePush(new(Push))
	cli.SetUnknownPush(CanetProcess)
	canetServer := "192.168.1.178:4001"
	log.Printf("connect to %s \r\n", canetServer)
	sess, stat := cli.Dial(canetServer, canetproto.CanetProtoFunc)
	if !stat.OK() {
		log.Printf("connect fail %v \r\n", stat)
		erpc.Fatalf("%v", stat)
	}
	log.Printf("connect success RemoteAddr=%s,LocalAddr=%s\r\n", sess.RemoteAddr().String(), sess.LocalAddr().String())

	//Sum：uint8
	// Com：09
	// AudioSource：（uint8）  0---室内播放语音；255(非0)----室外播放语音
	// AudioNo：（uint8）音频编号
	stat = sess.Push("38",
		[]byte{0, 0, 0x4, 0x09, 0xff, 0x1d, 0x4f}, //4f
	)
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	for {

		var key string
		fmt.Scan(&key)
		switch key {
		case "q":
			return
		case "s":
			//播放语音
			stat = sess.Push("38",
				[]byte{0, 0, 0x4, 0x09, 0xff, 0x1d, 0x4f}, //4f
			)
			if !stat.OK() {
				erpc.Fatalf("%v", stat)
			}
		case "r":
			//查询状态
			stat = sess.Push("38",
				[]byte{0, 0, 0x2, 0x0b, 0x33}, //33
			)
			if !stat.OK() {
				erpc.Fatalf("%v", stat)
			}
		}
		time.Sleep(time.Second * 1)
		erpc.Printf("Wait 1 seconds to receive the push...")
		//time.Sleep(time.Second * 60)
	}
}

// Push push handler
type Push struct {
	erpc.PushCtx
}

// Push handles '/push/status' message
func (p *Push) Status(arg *string) *erpc.Status {
	erpc.Printf("server status: %s", *arg)
	return nil
}

// Push push handler
type Canet struct {
	erpc.PushCtx
}

// Push handles '/Canet' message
func (p *Canet) Canet(arg *[]byte) *erpc.Status {
	erpc.Printf("Canet data: %v", *arg)
	return nil
}
func CanetProcess(upc erpc.UnknownPushCtx) *erpc.Status {
	fmt.Println("CanetProcess data receive:", upc.InputBodyBytes())
	return erpc.NewStatus(200, "ok")
}
