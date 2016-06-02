package log

import logging "github.com/op/go-logging"

var (
	log = logging.MustGetLogger("scheduler-shared")
)

func init() {
	log.ExtraCalldepth += 1
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Warningf(format string, args ...interface{}) {
	log.Warningf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func Warning(err error) {
	log.Warning(err.Error())
}

func Error(err error) {
	log.Error(err)
}

func Fatal(err error) {
	log.Fatal(err)
}
