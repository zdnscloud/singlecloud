package log4go

type BlackHole struct {
}

func NewBlackHole() *BlackHole {
	return &BlackHole{}
}

func (w *BlackHole) LogWrite(rec *LogRecord) {
}

func (w *BlackHole) Debug(fmt string, args ...interface{}) {}
func (w *BlackHole) Info(fmt string, args ...interface{})  {}
func (w *BlackHole) Warn(fmt string, args ...interface{}) error {
	return nil
}

func (w *BlackHole) Error(fmt string, args ...interface{}) error {
	return nil
}

func (w *BlackHole) Close() {
}
