package log

import (
	"os"
	"strconv"

	stack "github.com/Gurpartap/logrus-stack"
	formatter "github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	IsDebug bool
}

func New() (l *Logger) {
	txtformatter := &logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	}

	l = &Logger{Logger: logrus.New()}

	if l.IsDebug, _ = strconv.ParseBool(os.Getenv("DEBUG")); !l.IsDebug {
		logrus.SetFormatter(txtformatter)
	} else {
		l.AddHook(stack.StandardHook())
		l.SetLevel(logrus.DebugLevel)
		l.SetFormatter(&formatter.Formatter{
			ChildFormatter: txtformatter,
			Line:           true,
			Package:        true,
			File:           true,
		})
		l.Info("DEBUG=1")
	}
	return l
}
