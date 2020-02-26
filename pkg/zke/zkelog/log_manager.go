package zkelog

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	MaxLogSize = 100
)

type LogManager struct {
	lock     sync.Mutex
	watchers map[string]*LogWatcher
}

func New() *LogManager {
	return &LogManager{
		watchers: make(map[string]*LogWatcher),
	}
}

func (lm *LogManager) AddOrUpdate(name string, in <-chan string) {
	lm.lock.Lock()
	defer lm.lock.Unlock()
	lm.watchers[name] = newLogWatcher(in)
}

func (lm *LogManager) Delete(name string) error {
	lm.lock.Lock()
	defer lm.lock.Unlock()
	if _, ok := lm.watchers[name]; !ok {
		return fmt.Errorf("delete no exist cluster %s logger", name)
	}

	delete(lm.watchers, name)
	return nil
}

func (lm *LogManager) get(name string) *LogWatcher {
	lm.lock.Lock()
	defer lm.lock.Unlock()
	lw, ok := lm.watchers[name]
	if ok {
		return lw
	}
	return nil
}

type LogListener struct {
	lastID uint64
	stopCh chan struct{}
	logCh  chan string
}

func (l *LogListener) LogChannel() <-chan string {
	return l.logCh
}

func (l *LogListener) Stop() {
	l.stopCh <- struct{}{}
	<-l.stopCh
	close(l.logCh)
}

type ZKELog struct {
	id      uint64
	content string
}

type LogWatcher struct {
	logID   uint64
	lock    sync.RWMutex
	cond    *sync.Cond
	logList *list.List
	stopCh  chan struct{}
}

func (lw *LogWatcher) Stop() {
	close(lw.stopCh)
}

func newLogWatcher(in <-chan string) *LogWatcher {
	lw := &LogWatcher{
		logList: list.New(),
		stopCh:  make(chan struct{}),
	}
	lw.cond = sync.NewCond(&lw.lock)
	go func() {
		for {
			l, ok := <-in
			if !ok {
				return
			}
			lw.add(l)
		}
	}()
	return lw
}

func (lw *LogWatcher) add(in string) {
	id := atomic.AddUint64(&lw.logID, 1)
	l := ZKELog{
		id:      id,
		content: in,
	}
	lw.lock.Lock()
	lw.logList.PushBack(l)
	if uint(lw.logList.Len()) > MaxLogSize {
		elem := lw.logList.Front()
		lw.logList.Remove(elem)
	}
	lw.lock.Unlock()
	lw.cond.Broadcast()
}

func (lw *LogWatcher) addListener() *LogListener {
	l := &LogListener{
		lastID: 0,
		stopCh: make(chan struct{}),
		logCh:  make(chan string),
	}

	go lw.publishLog(l)
	return l
}

func (lw *LogWatcher) publishLog(l *LogListener) {
	batchSize := MaxLogSize / 4
	logs := make([]string, batchSize)
	for {
		lastID, c := lw.getLogsAfterID(l.lastID, logs)
		select {
		case <-l.stopCh:
			l.stopCh <- struct{}{}
			return
		default:
		}

		if c == 0 {
			lw.lock.Lock()
			lw.cond.Wait()
			lw.lock.Unlock()
			continue
		}

		l.lastID = lastID
		for i := 0; i < c; i++ {
			select {
			case <-l.stopCh:
				l.stopCh <- struct{}{}
				return
			case l.logCh <- logs[i]:
			}
		}
	}
}

func (lw *LogWatcher) getLogsAfterID(id uint64, logs []string) (uint64, int) {
	lw.lock.RLock()
	defer lw.lock.RUnlock()

	elem := lw.logList.Front()
	if elem == nil {
		return 0, 0
	}

	begID := elem.Value.(ZKELog).id
	if id < begID {
		return lw.getLogsFromOutdated(id, logs)
	}

	elem = lw.logList.Back()
	if elem == nil {
		return 0, 0
	}

	endID := elem.Value.(ZKELog).id
	if id == endID {
		return 0, 0
	}

	if id-begID < endID-id {
		return lw.getLogsFromOutdated(id, logs)
	} else {
		return lw.getLogsFromLatest(id, logs)
	}
}

func (lw *LogWatcher) getLogsFromOutdated(id uint64, logs []string) (uint64, int) {
	elem := lw.logList.Front()
	for elem.Value.(ZKELog).id <= id {
		elem = elem.Next()
	}
	return lw.getLogsFromElem(elem, logs)
}

func (lw *LogWatcher) getLogsFromLatest(id uint64, logs []string) (uint64, int) {
	elem := lw.logList.Back()
	for elem.Value.(ZKELog).id > id {
		elem = elem.Prev()
	}
	elem = elem.Next()
	return lw.getLogsFromElem(elem, logs)
}

func (lw *LogWatcher) getLogsFromElem(elem *list.Element, logs []string) (uint64, int) {
	lc := 0
	batch := len(logs)
	startID := elem.Value.(ZKELog).id
	for elem != nil && lc < batch {
		l := elem.Value.(ZKELog)
		logs[lc] = l.content
		lc += 1
		elem = elem.Next()
	}
	return startID + uint64(lc-1), lc
}
