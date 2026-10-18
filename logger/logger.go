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
	return logrus.AllLevels
}

func (h *AntiGlobalHook) Fire(entry *logrus.Entry) error {
	fmt.Fprintf(os.Stderr, "\n🚨🚨 DEVELOPER ERROR: GLOBAL LOGGER USED 🚨🚨\n")
	fmt.Fprintf(os.Stderr, "Message: \"%s\"\n", entry.Message)
	return nil
}

func init() {
	logrus.AddHook(&AntiGlobalHook{})
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
}

// New creates and returns a completely independent logger instance.
func New(appCfg configs.AppConfig, serverCfg configs.ServerConfig, lokiCfg configs.LokiConfig) *logrus.Logger {
	// Create an isolated logger instance, NOT the global standard logger.
	log := logrus.New()

	// Silence the console for this specific instance
	// log.SetOutput(io.Discard)
	log.SetFormatter(&logrus.JSONFormatter{})

	log.SetLevel(logrus.DebugLevel) // See everything in development; Dynamic ?

	// include file:line in every entry
	log.SetReportCaller(true)

	// Per-service Loki hook options
	opts := lokirus.NewLokiHookOptions().
		WithStaticLabels(lokirus.Labels{
			"app":          appCfg.Name,
			"service_name": serverCfg.Name,
			"env":          appCfg.Env,
		}).
		WithLevelMap(lokirus.LevelMap{
			logrus.PanicLevel: "critical",
		})

	hook := lokirus.NewLokiHookWithOpts(
		lokiCfg.Url,
		opts,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	)

	if hook != nil {
		log.AddHook(hook) // this instance only
	}

	return log
}
