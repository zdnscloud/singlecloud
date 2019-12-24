package alarm

func (ac *AlarmCache) AckChannel() <-chan int {
	go ac.publishAck()
	return ac.ackCh
}

func (ac *AlarmCache) publishAck() {
	num := ac.unAckNumber
	for {
		if ac.unAckNumber == 0 {
			ac.lock.Lock()
			ac.cond.Wait()
			ac.lock.Unlock()
			continue
		}
		select {
		case <-ac.stopCh:
			ac.stopCh <- struct{}{}
			return
		default:
			if num != ac.unAckNumber {
				ac.ackCh <- 1
				num = ac.unAckNumber
			}
		}
	}
}
