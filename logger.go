package libws

type logger interface {
	WithField(key string, value any) logger
	Debug(args ...any)
	Debugf(format string, args ...any)
	Debugln(args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Infoln(args ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
	Warnln(args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Errorln(args ...any)
}
