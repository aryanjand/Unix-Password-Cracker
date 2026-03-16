package controller

import (
	"sync"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)


type WorkerManager struct {
	sync.Mutex

	workers map[string]Worker
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

func (cm *WorkerManager) BroadcastMessage(msg protocol.Command) {
	cm.Lock()
	defer cm.Unlock()

	for id, worker := range cm.workers {
		worker.conn.Send <- protocol.Message{Command: msg}
		if msg == protocol.MsgStop {
			delete(cm.workers, id)
		}
	}
}
