package alarm

func (aw *AlarmCache) AckChannel() <-chan int {
	go aw.publishAck()
	return aw.ackCh
}

func (aw *AlarmCache) publishAck() {
	num := aw.unAckNumber
	for {
		if aw.unAckNumber == 0 {
			aw.lock.Lock()
			aw.cond.Wait()
			aw.lock.Unlock()
			continue
		}
		select {
		case <-aw.stopCh:
			aw.stopCh <- struct{}{}
			return
		default:
			if num != aw.unAckNumber {
				aw.ackCh <- 1
				num = aw.unAckNumber
			}
		}
	}
}
