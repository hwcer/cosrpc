package server

import (
	"testing"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosrpc"
)

// announceDummy 非空服务用的最小注册体：一个合法方法即可让表挂上路由
// （方法形状与 battle/service 同款：*cosrpc.Context 入参、单返回值 + Caller）。
type announceDummy struct{}

func (this *announceDummy) Caller(node *registry.Node, c *cosrpc.Context) interface{} {
	return nil
}

func (this *announceDummy) Ping(c *cosrpc.Context) interface{} {
	return nil
}

// TestCollectAnnounceSkipsEmptyService 空表不宣告、非空表带元数据。
//
// 场景出自 2026-10-21 战区服：非 game 进程 import yyds/context 后，包级
// server.Service("game") 预建的空 game 表被 startRegister 宣告进服务发现，
// /elf/game 里混入一个只会应答 can't find service game 的空节点。
// 收口防线放在宣告的唯一入口（collectAnnounce），空表一律不宣告。
func TestCollectAnnounceSkipsEmptyService(t *testing.T) {
	xs := New()
	t.Cleanup(func() { _ = xs.Close() })

	// 空表：只建表不挂路由（模拟非 game 进程里 yyds/context 的 game 表）。
	// 用 xs.Service（cosrpc 层入口，与 server.Service 同款）而非裸 Registry.Service，
	// 保持与生产路径同一套方法过滤器。
	_ = xs.Service("empty")
	// 非空表：挂一条路由（无前缀注册，与 battle/service 同款）
	nonEmpty := xs.Service("game")
	if err := nonEmpty.Register(&announceDummy{}); err != nil {
		t.Fatalf("注册非空服务失败: %v", err)
	}
	Metadata.Set("game", "sid=21")
	t.Cleanup(func() { delete(Metadata, "game") })

	got := xs.collectAnnounce()
	if len(got) != 1 {
		t.Fatalf("空表被宣告了: %v（want 只含 game）", got)
	}
	if v, ok := got["game"]; !ok || v != "sid=21" {
		t.Fatalf("非空服务的元数据丢了: %v", got)
	}
}
