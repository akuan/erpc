package main

import (
	"log"
	"time"

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

	cli := erpc.NewPeer(erpc.PeerConfig{Network: "tcp4", RedialTimes: 3, RedialInterval: time.Millisecond * 20, DialTimeout: time.Second * 5})
	defer cli.Close()
	//cli.SetTLSConfig(erpc.GenerateTLSConfigForClient())

	cli.RoutePush(new(Push))

	cli.SubRoute("/cli").
		RoutePush(new(Push))
	canetServer := "192.168.1.178:4001"
	log.Printf("connect to %s \r\n", canetServer)
	sess, stat := cli.Dial(canetServer)
	if !stat.OK() {
		log.Printf("connect fail %v \r\n", stat)
		erpc.Fatalf("%v", stat)
	}
	log.Printf("connect success RemoteAddr=%s,LocalAddr=%s\r\n", sess.RemoteAddr().String(), sess.LocalAddr().String())

	var result int
	stat = sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
		erpc.WithAddMeta("author", "andeya"),
	).Status()
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result: %d", result)
	erpc.Printf("Wait 10 seconds to receive the push...")
	time.Sleep(time.Second * 10)

	stat = sess.Call("/srv/math/v2/add_2",
		[]int{10, 20, 30, 40, 50},
		&result,
		erpc.WithSetMeta("push_status", "yes"),
	).Status()
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
