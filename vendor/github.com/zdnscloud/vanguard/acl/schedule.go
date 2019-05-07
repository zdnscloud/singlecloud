package acl

import (
	"errors"
	//"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/vanguard/config"
)

var (
	ErrTimeRangeInvalid  = errors.New("time range isn't valid")
	ErrHourMinuteInvalid = errors.New("hour minute format invalid")
	ErrWeekDayInvalid    = errors.New("weekday format invalid")
	ErrDayInvalid        = errors.New("day format invalid")
	ErrMonthInvalid      = errors.New("month format invalid")
	ErrDateInvalid       = errors.New("date format invalid")
)

const secondsOneDay = 24 * 3600 * time.Second

type TimeRange interface {
	IncludeTime(time.Time) bool
}

type Schedule struct {
	ranges []TimeRange
}

func newSchedule(timeRanges []config.TimeRange) (*Schedule, error) {
	var ranges []TimeRange
	for _, timeRangeConf := range timeRanges {
		timeRange, err := rangeBuilder(timeRangeConf.Begin, timeRangeConf.End)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, timeRange)
	}

	return &Schedule{ranges}, nil
}

func (s *Schedule) IncludeTime(t time.Time) bool {
	for _, timeRange := range s.ranges {
		if timeRange.IncludeTime(t) {
			return true
		}
	}
	return false
}

//"15:04, 12:00"
//"1 15:04, 2 12:00"
//"1 2 13:00, 1 2 14:00"
func rangeBuilder(from, to string) (TimeRange, error) {
	fromRange := strings.Split(from, " ")
	toRange := strings.Split(to, " ")
	if len(fromRange) != len(toRange) {
		return nil, ErrTimeRangeInvalid
	}

	switch len(fromRange) {
	case 1:
		return newPeriodicInHour(fromRange[0], toRange[0])
	case 2:
		return newPeriodicInWeekDay(fromRange[0], fromRange[1], toRange[0], toRange[1])
	case 3:
		return newPeriodicInDate(fromRange[0], fromRange[1], fromRange[2], toRange[0], toRange[1], toRange[2])
	default:
		return nil, ErrTimeRangeInvalid
	}
}

type PeriodicInHour struct {
	startHour, endHour     int
	startMinute, endMinute int
	expandDay              bool
}

func newPeriodicInHour(from, to string) (TimeRange, error) {
	startHour, startMinute, err := hourMinFromString(from)
	if err != nil {
		return nil, err
	}
	endHour, endMinute, err := hourMinFromString(to)
	if err != nil {
		return nil, err
	}

	if startHour == endHour {
		if startMinute > endMinute {
			return nil, ErrHourMinuteInvalid
		}
	}

	return &PeriodicInHour{
		startHour:   startHour,
		endHour:     endHour,
		startMinute: startMinute,
		endMinute:   endMinute,
		expandDay:   startHour > endHour,
	}, nil
}

func (p *PeriodicInHour) IncludeTime(t time.Time) bool {
	year, month, day, loc := t.Year(), t.Month(), t.Day(), t.Location()
	start := time.Date(year, month, day, p.startHour, p.startMinute, 0, 0, loc)
	end := time.Date(year, month, day, p.endHour, p.endMinute, 0, 0, loc)
	if p.expandDay {
		return t.After(start) || t.Before(end)
	} else {
		return t.After(start) && t.Before(end)
	}
}

type PeriodicInWeekDay struct {
	startWeekDay, endWeekDay time.Weekday
	startHour, endHour       int
	startMinute, endMinute   int
	expandWeek               bool
}

func newPeriodicInWeekDay(fromDay, fromHourMinute string, toDay, toHourMinute string) (TimeRange, error) {
	startHour, startMinute, err := hourMinFromString(fromHourMinute)
	if err != nil {
		return nil, err
	}
	endHour, endMinute, err := hourMinFromString(toHourMinute)
	if err != nil {
		return nil, err
	}
	startWeekDay, err := weekDayFromString(fromDay)
	if err != nil {
		return nil, err
	}
	endWeekDay, err := weekDayFromString(toDay)
	if err != nil {
		return nil, err
	}

	if startWeekDay == endWeekDay {
		if startHour > endHour || (startHour == endHour && startMinute > endMinute) {
			return nil, ErrWeekDayInvalid
		}
	}

	return &PeriodicInWeekDay{
		startWeekDay: startWeekDay,
		endWeekDay:   endWeekDay,
		startHour:    startHour,
		endHour:      endHour,
		startMinute:  startMinute,
		endMinute:    endMinute,
		expandWeek:   startWeekDay > endWeekDay,
	}, nil
}

func (p *PeriodicInWeekDay) IncludeTime(t time.Time) bool {
	weekDay := t.Weekday()
	year, month, day, loc := t.Year(), t.Month(), t.Day(), t.Location()
	start_ := time.Date(year, month, day, p.startHour, p.startMinute, 0, 0, loc)
	end_ := time.Date(year, month, day, p.endHour, p.endMinute, 0, 0, loc)
	start := start_.Add(time.Duration(p.startWeekDay-weekDay) * secondsOneDay)
	end := end_.Add(time.Duration(p.endWeekDay-weekDay) * secondsOneDay)
	if p.expandWeek {
		return t.After(start) || t.Before(end)
	} else {
		return t.After(start) && t.Before(end)
	}
}

type PeriodicInDate struct {
	startMonth, endMonth   time.Month
	startDay, endDay       int
	startHour, endHour     int
	startMinute, endMinute int
	expandYear             bool
}

func newPeriodicInDate(fromMonth, fromDay, fromHourMinute string, toMonth, toDay, toHourMinute string) (TimeRange, error) {
	startMonth, err := monthFromString(fromMonth)
	if err != nil {
		return nil, err
	}
	endMonth, err := monthFromString(toMonth)
	if err != nil {
		return nil, err
	}

	startHour, startMinute, err := hourMinFromString(fromHourMinute)
	if err != nil {
		return nil, err
	}
	endHour, endMinute, err := hourMinFromString(toHourMinute)
	if err != nil {
		return nil, err
	}
	startDay, err := dayFromString(fromDay)
	if err != nil {
		return nil, err
	}
	endDay, err := dayFromString(toDay)
	if err != nil {
		return nil, err
	}

	if startMonth <= endMonth {
		now := time.Now()
		year, loc := now.Year(), now.Location()
		start := time.Date(year, startMonth, startDay, startHour, startMinute, 0, 0, loc)
		end := time.Date(year, endMonth, endDay, endHour, endMinute, 0, 0, loc)
		if end.Before(start) {
			return nil, ErrDateInvalid
		}
	}

	return &PeriodicInDate{
		startMonth:  startMonth,
		endMonth:    endMonth,
		startDay:    startDay,
		endDay:      endDay,
		startHour:   startHour,
		endHour:     endHour,
		startMinute: startMinute,
		endMinute:   endMinute,
		expandYear:  startMonth > endMonth,
	}, nil
}

func (p *PeriodicInDate) IncludeTime(t time.Time) bool {
	year, loc := t.Year(), t.Location()
	start := time.Date(year, p.startMonth, p.startDay, p.startHour, p.startMinute, 0, 0, loc)
	end := time.Date(year, p.endMonth, p.endDay, p.endHour, p.endMinute, 0, 0, loc)
	if p.expandYear {
		return t.After(start) || t.Before(end)
	} else {
		return t.After(start) && t.Before(end)
	}
}

func hourMinFromString(hm string) (int, int, error) {
	hms := strings.Split(hm, ":")
	if len(hms) != 2 {
		return 0, 0, ErrHourMinuteInvalid
	}

	h, err := strconv.Atoi(hms[0])
	if err != nil || h < 0 || h > 23 {
		return 0, 0, ErrHourMinuteInvalid
	}

	m, err := strconv.Atoi(hms[1])
	if err != nil || m < 0 || m > 59 {
		return 0, 0, ErrHourMinuteInvalid
	}

	return h, m, nil
}

func weekDayFromString(day string) (time.Weekday, error) {
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return 0, ErrWeekDayInvalid
	}

	weekDay := time.Weekday(dayInt)
	if weekDay < time.Sunday || weekDay > time.Saturday {
		return 0, ErrWeekDayInvalid
	} else {
		return weekDay, nil
	}
}

func dayFromString(day string) (int, error) {
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return 0, ErrDayInvalid
	}

	if dayInt < 1 || dayInt > 31 {
		return 0, ErrWeekDayInvalid
	} else {
		return dayInt, nil
	}
}

func monthFromString(monthString string) (time.Month, error) {
	monthInt, err := strconv.Atoi(monthString)
	if err != nil {
		return 0, ErrWeekDayInvalid
	}

	month := time.Month(monthInt)
	if month < time.January || month > time.December {
		return 0, ErrMonthInvalid
	} else {
		return month, nil
	}
}
