package kone

import (
	"github.com/nxsre/lumberjack"
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("kone")

func InitLogger(debug bool) {
	format := logging.MustStringFormatter(
		`%{color}%{time:06-01-02 15:04:05.000} %{level:.4s} @%{shortfile}%{color:reset} %{message}`,
	)
	logging.SetFormatter(format)
	logging.SetBackend(logging.NewLogBackend(&lumberjack.Logger{
		Filename:   "kone.log",
		TimeFormat: "2006-01-02T15",
		MaxSize:    -1,
		MaxAge:     1,
		MaxBackups: 10,
		LocalTime:  true,
		Compress:   true,
	}, "", 0))

	if debug {
		logging.SetLevel(logging.DEBUG, "kone")
	} else {
		logging.SetLevel(logging.INFO, "kone")
	}
}

func GetLogger() *logging.Logger {
	return logger
}
