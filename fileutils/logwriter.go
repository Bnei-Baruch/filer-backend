package fileutils

import (
	"os"
)

// LogWriter

func NewLogWriter(ctx LogCtx) *LogWriter {
	var err error

	if ctx.Path == "" {
		ctx.out = os.Stderr
	} else {
		ctx.out, err = os.OpenFile(ctx.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			ctx.out = os.Stderr
		}
	}

	return &LogWriter{
		ctx: ctx,
	}
}

func (lw *LogWriter) Close() (err error) {
	if lw.ctx.out == nil || lw.ctx.out == os.Stderr {
		err = nil
	} else {
		err = lw.ctx.out.Close()
		lw.ctx.out = nil
	}
	return
}

func (lw *LogWriter) Write(b []byte) (n int, err error) {
	n, err = lw.ctx.out.Write(b)
	return
}
