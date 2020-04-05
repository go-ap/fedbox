package badger

type logger struct {
	logFn loggerFn
	errFn loggerFn
}

func (l logger) Errorf(s string, p ...interface{}) {
	if l.errFn == nil {
		return
	}
	l.errFn(nil, s, p...)
}
func (l logger) Warningf(s string, p ...interface{}) {
	if l.logFn == nil {
		return
	}
	l.logFn(nil, s, p...)
}
func (l logger) Infof(s string, p ...interface{}) {
	if l.logFn == nil {
		return
	}
	l.logFn(nil, s, p...)
}
func (l logger) Debugf(s string, p ...interface{}) {
	if l.logFn == nil {
		return
	}
	l.logFn(nil, s, p...)
}
