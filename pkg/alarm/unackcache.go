package alarm

func (ac *AlarmCache) publishAck(al *AlarmListener) {
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
				al.alarmCh <- ac.unAckNumber
				num = ac.unAckNumber
			}
		}
	}
}
