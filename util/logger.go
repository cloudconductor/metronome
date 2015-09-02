package util

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type LogFormatter struct {
}

//	Set time format and level in log format
func (f *LogFormatter) Format(entry *log.Entry) ([]byte, error) {
	time := entry.Time.Format("2006-01-02T15:04:05.000Z07:00")
	level := strings.ToUpper(entry.Level.String())
	if level == "WARNING" {
		level = "WARN"
	}
	return []byte(fmt.Sprintf("%s [%-5s] %s\n", time, level, entry.Message)), nil
}
