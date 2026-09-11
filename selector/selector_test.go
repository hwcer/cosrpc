package selector

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/smallnest/rpcx/share"
)

// TestSelectBasic 基础选路:按最小负载选节点,SelectWithServerId 按 sid 过滤
func TestSelectBasic(t *testing.T) {
	s := New("test")
	s.UpdateServer(map[string]string{
		"tcp@127.0.0.1:8001": "_rpc_srv_sid=a&_rpc_srv_avg=10",
		"tcp@127.0.0.1:8002": "_rpc_srv_sid=b&_rpc_srv_avg=1",
	})
	ctx := context.WithValue(context.Background(), share.ReqMetaDataKey, map[string]string{})
	if got := s.Select(ctx, "path", "method", nil); got != "tcp@127.0.0.1:8002" {
		t.Fatalf("应选最小负载 8002,拿到 %q", got)
	}
	ctx2 := context.WithValue(context.Background(), share.ReqMetaDataKey,
		map[string]string{MetaDataServerId: "a"})
	if got := s.Select(ctx2, "path", "method", nil); got != "tcp@127.0.0.1:8001" {
		t.Fatalf("sid 过滤应选 8001,拿到 %q", got)
	}
	// 固定地址直通
	ctx3 := context.WithValue(context.Background(), share.ReqMetaDataKey,
		map[string]string{MetaDataAddress: "tcp@10.0.0.9:1234"})
	if got := s.Select(ctx3, "path", "method", nil); got != "tcp@10.0.0.9:1234" {
		t.Fatalf("固定地址必须直通,拿到 %q", got)
	}
}

// TestUpdateServerKeepsNodes 服务下线重建后 index 延续、Average 以发现侧为准
func TestUpdateServerKeepsNodes(t *testing.T) {
	s := New("test")
	s.UpdateServer(map[string]string{"tcp@127.0.0.1:8001": "_rpc_srv_sid=a&_rpc_srv_avg=5"})
	snapshot := s.snapshot.Load()
	old := snapshot.all["tcp@127.0.0.1:8001"]
	old.Average.Add(3) //模拟本地负载累计
	old.index = 7

	//重建:同地址节点应延续 index;Average 来自发现侧元数据(5),不继承本地累计
	s.UpdateServer(map[string]string{"tcp@127.0.0.1:8001": "_rpc_srv_sid=a&_rpc_srv_avg=5"})
	fresh := s.snapshot.Load().all["tcp@127.0.0.1:8001"]
	if fresh == old {
		t.Fatal("UpdateServer 必须重建节点")
	}
	if fresh.index != 7 {
		t.Fatalf("index 必须延续,拿到 %d", fresh.index)
	}
	if v := fresh.Average.Load(); v != 5 {
		t.Fatalf("Average 应取发现侧的 5,拿到 %d", v)
	}
}

// TestSelectConcurrentWithUpdate 🔴判据回归:Select(请求协程)与 UpdateServer
// (发现 watch 协程)并发,曾有裸字段读/COW 无原子发布的竞争,-race 下必须干净
func TestSelectConcurrentWithUpdate(t *testing.T) {
	s := New("test")
	s.UpdateServer(map[string]string{
		"tcp@127.0.0.1:8001": "_rpc_srv_sid=a&_rpc_srv_avg=1",
		"tcp@127.0.0.1:8002": "_rpc_srv_sid=b&_rpc_srv_avg=2",
	})
	done := make(chan struct{})
	var wg sync.WaitGroup
	//8 个请求协程持续选路
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), share.ReqMetaDataKey, map[string]string{})
			for {
				select {
				case <-done:
					return
				default:
					if addr := s.Select(ctx, "path", "method", nil); addr == "" {
						t.Error("在线服务 Select 不得返回空地址")
						return
					}
				}
			}
		}()
	}
	//2 个发现协程持续整表更新(含增删,制造 all/services 重建)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				servers := map[string]string{
					fmt.Sprintf("tcp@127.0.0.1:%d", 8001+j%4): "_rpc_srv_sid=a&_rpc_srv_avg=1",
				}
				if j%2 == 0 {
					servers["tcp@127.0.0.1:9000"] = fmt.Sprintf("_rpc_srv_sid=b&_rpc_srv_avg=%d", j)
				}
				s.UpdateServer(servers)
			}
		}(i)
	}
	go func() { close(done); }()
	wg.Wait()
}
