package logcfg

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"runtime"
)

// RunLoggerConfig производит настройку logrus устанавливая уровень логирования,
// формат логируемой информации и настройки записи логов в файл.
func RunLoggerConfig(EnvLogs string) {

	logLevel, err := logrus.ParseLevel(EnvLogs)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(logLevel)
	logrus.SetReportCaller(true)

	//Настраиваем формат логируемой информации
	logrus.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (function string, file string) {
			_, filename := path.Split(f.File)
			filename = fmt.Sprintf("%s.%d.%s", filename, f.Line, f.Function)
			return "", filename
		},
	})
	// Настраиваем запись логов в файл
	mw := io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename:   "tgBot.log",
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     30,
	})
	logrus.SetOutput(mw)
}
