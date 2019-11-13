package ignorecase_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/tp/v6"
	"github.com/henrylee2cn/tp/v6/plugin/ignorecase"
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *tp.Status) {
	h.Session().Push("/push/test", map[string]string{
		"your_id": string(h.PeekMeta("peer_id")),
	})
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

func TestIngoreCase(t *testing.T) {
	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090, Network: "quic"}, ignorecase.NewIgnoreCase())
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{Network: "quic"}, ignorecase.NewIgnoreCase())
	cli.RoutePush(new(Push))
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	stat = sess.Call("/home/TesT",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		tp.WithAddMeta("peer_id", "110"),
	).Status()
	if !stat.OK() {
		t.Error(stat)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}

type Push struct {
	tp.PushCtx
}

func (p *Push) Test(arg *map[string]string) *tp.Status {
	tp.Infof("receive push(%s):\narg: %#v\n", p.IP(), arg)
	return nil
}
