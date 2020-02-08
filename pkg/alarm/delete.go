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
	//dd, _ := time.ParseDuration("-2m")
	startTime := t.Add(dd)
	firstAlarm, err := getAlarmFromDB(ac.alarmsTable, ac.firstID)
	if err != nil {
		return delNumber, err
	}
	if startTime.After(time.Time(firstAlarm.Time)) {
		for {
			firstAlarm, err := getAlarmFromDB(ac.alarmsTable, ac.firstID)
			if err != nil {
				return delNumber, err
			}
			if startTime.After(time.Time(firstAlarm.Time)) {
				if err := ac.Del(firstAlarm.UID); err != nil {
					return delNumber, err
				}
				if !firstAlarm.Acknowledged {
					delNumber += 1
				}
			} else {
				break
			}
		}
	} else {
		if err := ac.Del(firstAlarm.UID); err != nil {
			return delNumber, err
		}
		if !firstAlarm.Acknowledged {
			delNumber += 1
		}
	}
	return delNumber, nil
}
