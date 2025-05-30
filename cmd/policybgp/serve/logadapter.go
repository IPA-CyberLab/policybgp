package serve

import (
	gobgplog "github.com/osrg/gobgp/v4/pkg/log"

	"go.uber.org/zap"
)

type logAdapter struct {
	l *zap.SugaredLogger
}

var _ gobgplog.Logger = &logAdapter{}

func (la *logAdapter) Panic(msg string, fields gobgplog.Fields) {
	la.l.Panicw(msg, fieldsToKeyvals(fields)...)
}

func (la *logAdapter) Fatal(msg string, fields gobgplog.Fields) {
	la.l.Fatalw(msg, fieldsToKeyvals(fields)...)
}

func (la *logAdapter) Error(msg string, fields gobgplog.Fields) {
	la.l.Errorw(msg, fieldsToKeyvals(fields)...)
}

func (la *logAdapter) Warn(msg string, fields gobgplog.Fields) {
	la.l.Warnw(msg, fieldsToKeyvals(fields)...)
}

func (la *logAdapter) Info(msg string, fields gobgplog.Fields) {
	la.l.Infow(msg, fieldsToKeyvals(fields)...)
}

func (la *logAdapter) Debug(msg string, fields gobgplog.Fields) {
	la.l.Debugw(msg, fieldsToKeyvals(fields)...)
}

func (la *logAdapter) SetLevel(level gobgplog.LogLevel) {
	// Note: zap doesn't support runtime level changes easily
	// This is a placeholder implementation
}

func (la *logAdapter) GetLevel() gobgplog.LogLevel {
	// Note: zap doesn't provide easy level introspection
	// Return a default level
	return gobgplog.InfoLevel
}

func fieldsToKeyvals(fields gobgplog.Fields) []interface{} {
	keyvals := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		keyvals = append(keyvals, k, v)
	}
	return keyvals
}
