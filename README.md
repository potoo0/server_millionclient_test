参考:
- https://github.com/eranyanay/1m-go-websockets

## gnet

gnet build tags:
- poll_opt. 源码为: *pkg/netpoll/poller_epoll_ultimate.go*, 否则为: *pkg/netpoll/poller_epoll_default.go*
- gc_opt. 源码为: *conn_matrix.go*(使用二维数组存储 conn, fd2gfd 变量存储二维数组的坐标), 否则为: *pkg/gnet/gnet.go*

gnet 思路:
1. Multicore 将会取 `runtime.NumCPU()` 作为 *numEventLoop*
2. 创建 *numEventLoop* 个 eventloop, 每个 eventloop 包含的
   1. Poller. 源码位置参考 poll_opt
   2. listeners. 如果是 ReusePort 则重新初始化 listeners, 否则使用 `engine.listeners`
   3. 在 `engine.concurrency` 启动的 goroutine 中调用 `run`(ReusePort=true) / `orbit`
3. eventloop 的 `run/orbit` 调用 `Poller.Polling` (阻塞)等待 epoll 事件, 每次最多取 `InitPollEventsCap = 128`, 未取到或者 unix.EINTR 后 `runtime.Gosched()` 挂起当前 P, 唤醒后继续等待
4. `Poller.Polling` 取到其事件后, 依次处理 `Poller.urgentAsyncTaskQueue` 中的任务, 然后处理(最多处理 `MaxAsyncTasksAtOneTime = 256`) `Poller.asyncTaskQueue` 中的任务
   1. task 任务从 `pkg/queue/queue.go`(底层 `sync.Pool`) 对象缓存池里取 
5. `gnet.Conn.AsyncWrite` 是调用 `Poller.Trigger` 将 conn 加入到 `Poller.urgentAsyncTaskQueue` 中

### 最佳实践

1. 耗时的阻塞推荐使用 workerPool:
   ```go
   // OnTraffic
   // ...
   s.pool.Submit(func() {
       time.Sleep(100 * time.Millisecond)
       conn.AsyncWrite(bytes, func(c gnet.Conn, err error) error {
           if err != nil {
               Logger.Info("write failed", zap.Error(err))
           }
           return err
       })
   })
   ```
