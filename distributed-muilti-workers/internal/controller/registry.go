package controller

import (
	"sync"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type WorkerManager struct {
	sync.Mutex

	workers    map[string]Worker
	shutdownWG sync.WaitGroup
}

func NewWorkerManger() *WorkerManager {
	return &WorkerManager{
		workers: make(map[string]Worker),
	}
}

func (cm *WorkerManager) Count() int {
	cm.Lock()
	defer cm.Unlock()

	return len(cm.workers)
}

func (cm *WorkerManager) AddWorker(id string, worker Worker) {
	cm.Lock()
	defer cm.Unlock()

	cm.workers[id] = worker
	cm.shutdownWG.Add(1)

	go func(id string, worker Worker) {
		defer cm.shutdownWG.Done()
		<-worker.conn.Stop.Done()
		cm.RemoveWorker(id)
		worker.logger.Printf("worker removed from manager (id=%s, active_workers=%d)", id, cm.Count())
	}(id, worker)
}

func (cm *WorkerManager) RemoveWorker(id string) {
	cm.Lock()
	defer cm.Unlock()

	delete(cm.workers, id)
}

func (cm *WorkerManager) GetWorker(id string) (Worker, bool) {
	cm.Lock()
	defer cm.Unlock()

	worker, ok := cm.workers[id]
	return worker, ok
}

// WaitForShutdown blocks until every worker that was ever added has had its
// connection torn down (see AddWorker), or until timeout elapses. It
// returns true if every worker finished cleanly within timeout.
func (cm *WorkerManager) WaitForShutdown(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		cm.shutdownWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (cm *WorkerManager) BroadcastMessage(msg protocol.Command) {
	cm.Lock()
	workers := make([]Worker, 0, len(cm.workers))
	for id, worker := range cm.workers {
		workers = append(workers, worker)
		if msg == protocol.MsgStop {
			delete(cm.workers, id)
		}
	}
	cm.Unlock()

	for _, worker := range workers {
		if err := worker.conn.SendMsg(protocol.Message{Command: msg}); err != nil {
			worker.logger.Printf("broadcast %s failed: %v", msg, err)
		}
	}
}
