package logger

import (
	"fmt"
	// "io"
	"os"

	"github.com/culionbear/lokirus"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/sirupsen/logrus"
)

// AntiGlobalHook intercepts any calls to the global logrus instance
type AntiGlobalHook struct{}

func (h *AntiGlobalHook) Levels() []logrus.Level {
	return logrus.AllLevels // Catch every single log level
}

func (h *AntiGlobalHook) Fire(entry *logrus.Entry) error {
	fmt.Fprintf(os.Stderr, "\n🚨🚨 DEVELOPER ERROR: GLOBAL LOGGER USED 🚨🚨\n")
	fmt.Fprintf(os.Stderr, "Message: \"%s\"\n", entry.Message)
	return nil
}

// init runs exactly once when this package is imported anywhere in your app.
func init() {
	logrus.AddHook(&AntiGlobalHook{})
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
}

// New creates and returns a completely independent logger instance.
func New(configs *configs.Configs) *logrus.Logger {
	// Create an isolated logger instance, NOT the global standard logger.
	log := logrus.New()

	// Silence the console for this specific instance
	// log.SetOutput(io.Discard)
	log.SetFormatter(&logrus.JSONFormatter{})

	log.SetLevel(logrus.DebugLevel) // See everything in development; Dynamic ?

	// automatic caller injection
	log.SetReportCaller(true)

	// Configure Loki options specific to THIS service
	opts := lokirus.NewLokiHookOptions().
		WithStaticLabels(lokirus.Labels{
			"app":          configs.App.App.Name,
			"service_name": configs.Server.ServerName,
			"env":          configs.App.App.Env,
		}).
		WithLevelMap(lokirus.LevelMap{
			logrus.PanicLevel: "critical",
		})

	hook := lokirus.NewLokiHookWithOpts(
		configs.App.Loki.Url,
		opts,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	)

	if hook != nil {
		log.AddHook(hook) // Only hooks onto THIS instance
	}

	return log
}
