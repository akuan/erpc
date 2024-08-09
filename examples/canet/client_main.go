package main

import (
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
		Network: "tcp4", RedialTimes: 3, RedialInterval: time.Millisecond * 20, DialTimeout: time.Second * 5,
		DefaultBodyCodec: "canet",
	})
	defer cli.Close()
	//cli.SetTLSConfig(erpc.GenerateTLSConfigForClient())

	cli.RoutePush(new(Push))
	cli.SubRoute("/cli").RoutePush(new(Push))

	canetServer := "192.168.1.178:4001"
	log.Printf("connect to %s \r\n", canetServer)
	sess, stat := cli.Dial(canetServer, canetproto.CanetProtoFunc())
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
		[]byte{0x27, 0x09, 1, 29},
	)
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("Wait 10 seconds to receive the push...")
	time.Sleep(time.Second * 10)
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
