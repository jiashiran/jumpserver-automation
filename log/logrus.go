package log

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger = logrus.New()

func init() {
	// Log as JSON instead of the default ASCII formatter.
	Logger.SetFormatter(&logrus.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	f, _ := os.OpenFile("logrus.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	_ = f

	Logger.SetOutput(f)

	// Only log the warning severity or above.
	Logger.SetLevel(logrus.InfoLevel)

}
