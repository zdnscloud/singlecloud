package alarm

import (
	"time"
)

func (ac *AlarmCache) delOver() (int, error) {
	var delNumber int
	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Local()
	dd, _ := time.ParseDuration("-720h")
	//t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, now.Location()).Local()
	//dd, _ := time.ParseDuration("-20m")
	startTime := t.Add(dd)

	if !ac.alarms[uintToStr(ac.firstID)].Acknowledged {
		delNumber += 1
	}
	if err := ac.Del(ac.firstID); err != nil {
		return delNumber, err
	}
	for {
		alarm := *ac.alarms[uintToStr(ac.firstID)]
		if startTime.Before(time.Time(alarm.Time)) {
			break
		}
		if err := ac.Del(ac.firstID); err != nil {
			return delNumber, err
		}
		if !alarm.Acknowledged {
			delNumber += 1
		}
	}
	return delNumber, nil
}
