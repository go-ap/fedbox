// +build storage_badger storage_all

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
	l.logFn(nil, "WARN: "+s, p...)
}
func (l logger) Infof(s string, p ...interface{}) {
	//if l.logFn == nil {
	//	return
	//}
	//l.logFn(nil, "INFO: " + s, p...)
}
func (l logger) Debugf(s string, p ...interface{}) {
	//if l.logFn == nil {
	//	return
	//}
	//l.logFn(nil, "DEBUG: " + s, p...)
}
