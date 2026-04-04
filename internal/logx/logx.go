package logx

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var enabledTopics = map[string]struct{}{}
var debugEnabled bool
var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelError,
}))

func Init(debug bool, topics []string) {
	enabledTopics = parseTopics(topics)
	debugEnabled = debug || envEnabled("PIKPAKCLI_DEBUG") || len(enabledTopics) > 0

	level := slog.LevelError
	if debugEnabled {
		level = slog.LevelDebug
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
}

func Enabled(topic string) bool {
	if !debugEnabled {
		return false
	}
	if topic == "" {
		return true
	}
	if _, ok := enabledTopics["all"]; ok {
		return true
	}
	_, ok := enabledTopics[topic]
	return ok
}

func Debug(topic string, args ...any) {
	if Enabled(topic) {
		logger.Debug(fmt.Sprint(args...))
	}
}

func Debugln(topic string, args ...any) {
	if Enabled(topic) {
		logger.Debug(fmt.Sprintln(args...))
	}
}

func Warn(topic string, args ...any) {
	if Enabled(topic) {
		logger.Warn(fmt.Sprint(args...))
	}
}

func Warnf(topic, format string, args ...any) {
	if Enabled(topic) {
		logger.Warn(fmt.Sprintf(format, args...))
	}
}

func Error(args ...any) {
	logger.Error(fmt.Sprint(args...))
}

func Errorf(format string, args ...any) {
	logger.Error(fmt.Sprintf(format, args...))
}

func parseTopics(topics []string) map[string]struct{} {
	res := map[string]struct{}{}
	for _, topic := range topics {
		for _, item := range strings.Split(topic, ",") {
			item = strings.TrimSpace(strings.ToLower(item))
			if item == "" {
				continue
			}
			res[item] = struct{}{}
		}
	}
	envTopics := strings.Split(os.Getenv("PIKPAKCLI_DEBUG_TOPICS"), ",")
	for _, item := range envTopics {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		res[item] = struct{}{}
	}
	if envEnabled("PIKPAKCLI_DEBUG") {
		res["all"] = struct{}{}
	}
	return res
}

func envEnabled(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on", "debug", "all":
		return true
	default:
		return false
	}
}
