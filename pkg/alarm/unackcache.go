package alarm

func (ac *AlarmCache) publishAck(al *AlarmListener) {
	for {
		num := ac.getUnAckNumber()

		if al.unAck == num {
			ac.lock.Lock()
			ac.cond.Wait()
			ac.lock.Unlock()
			continue
		}
		al.unAck = num
		select {
		case <-ac.stopCh:
			ac.stopCh <- struct{}{}
			return
		default:
			al.alarmCh <- num
		}
	}
}
