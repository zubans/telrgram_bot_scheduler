package parser

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"
    "time"
)

type EventEntry struct {
    Date        time.Time
    Description string
    RawDate     string
    IsValid     bool
}

var monthNames = map[string]int{
    "января":   1,
    "февраля":  2,
    "марта":    3,
    "апреля":   4,
    "мая":      5,
    "июня":     6,
    "июля":     7,
    "августа":  8,
    "сентября": 9,
    "октября":  10,
    "ноября":   11,
    "декабря":  12,
}

func ParseEventList(text string) []*EventEntry {
    lines := strings.Split(text, "\n")
    var events []*EventEntry

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        event := parseEventLine(line)
        if event != nil {
            events = append(events, event)
        }
    }

    return events
}

func parseEventLine(line string) *EventEntry {
    entry := &EventEntry{
        RawDate:     line,
        IsValid:     false,
    }

    var date time.Time
    var description string
    var found bool

    date, description, found = tryParseRussianFormat(line)
    if !found {
        date, description, found = tryParseDotFormat(line)
    }
    if !found {
        date, description, found = tryParseRangeFormat(line)
    }

    if found {
        entry.Date = date
        entry.Description = description
        entry.IsValid = true
    }

    return entry
}

func tryParseRussianFormat(line string) (time.Time, string, bool) {
    re := regexp.MustCompile(`^(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)\s+(.*)$`)
    matches := re.FindStringSubmatch(line)

    if len(matches) < 4 {
        return time.Time{}, "", false
    }

    day, err := strconv.Atoi(matches[1])
    if err != nil || day < 1 || day > 31 {
        return time.Time{}, "", false
    }

    month, ok := monthNames[matches[2]]
    if !ok {
        return time.Time{}, "", false
    }

    description := strings.TrimSpace(matches[3])

    now := time.Now()
    year := now.Year()

    dateToCheck := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
    if dateToCheck.Before(now) && month < int(now.Month()) {
        year++
        dateToCheck = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
    }

    return dateToCheck, description, true
}

func tryParseDotFormat(line string) (time.Time, string, bool) {
    re := regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\s+(.*)$`)
    matches := re.FindStringSubmatch(line)

    if len(matches) < 4 {
        return time.Time{}, "", false
    }

    day, err := strconv.Atoi(matches[1])
    if err != nil || day < 1 || day > 31 {
        return time.Time{}, "", false
    }

    month, err := strconv.Atoi(matches[2])
    if err != nil || month < 1 || month > 12 {
        return time.Time{}, "", false
    }

    description := strings.TrimSpace(matches[3])

    now := time.Now()
    year := now.Year()

    dateToCheck := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
    if dateToCheck.Before(now) {
        year++
        dateToCheck = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
    }

    return dateToCheck, description, true
}

func tryParseRangeFormat(line string) (time.Time, string, bool) {
    re := regexp.MustCompile(`^(\d{1,2})-(\d{1,2})\.(\d{1,2})\s+(.*)$`)
    matches := re.FindStringSubmatch(line)

    if len(matches) < 5 {
        return time.Time{}, "", false
    }

    startDay, err := strconv.Atoi(matches[1])
    if err != nil || startDay < 1 || startDay > 31 {
        return time.Time{}, "", false
    }

    month, err := strconv.Atoi(matches[3])
    if err != nil || month < 1 || month > 12 {
        return time.Time{}, "", false
    }

    description := strings.TrimSpace(matches[4])

    now := time.Now()
    year := now.Year()

    dateToCheck := time.Date(year, time.Month(month), startDay, 0, 0, 0, 0, time.Local)
    if dateToCheck.Before(now) {
        year++
        dateToCheck = time.Date(year, time.Month(month), startDay, 0, 0, 0, 0, time.Local)
    }

    return dateToCheck, description, true
}

func GetUpcomingEvents(events []*EventEntry, daysAhead int) []*EventEntry {
    var upcoming []*EventEntry
    now := time.Now()
    targetDate := now.AddDate(0, 0, daysAhead)

    for _, event := range events {
        if !event.IsValid {
            continue
        }
        if (event.Date.Equal(now) || event.Date.After(now)) && event.Date.Before(targetDate.AddDate(0, 0, 1)) {
            upcoming = append(upcoming, event)
        }
    }

    return upcoming
}

func FormatEventForMessage(event *EventEntry) string {
    if !event.IsValid {
        return ""
    }
    dateStr := event.Date.Format("02 January")
    return fmt.Sprintf("📅 %s - %s", dateStr, event.Description)
}
