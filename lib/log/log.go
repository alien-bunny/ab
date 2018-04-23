// Copyright 2018 Tamás Demeter-Haludka
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"

	"github.com/fatih/color"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-logfmt/logfmt"
)

type Logger = log.Logger
type LoggerFunc = log.LoggerFunc
type Option = level.Option

func With(logger Logger, keyvals ...interface{}) Logger {
	return log.With(logger, keyvals...)
}

func WithPrefix(logger Logger, keyvals ...interface{}) Logger {
	return log.WithPrefix(logger, keyvals...)
}

var _ Logger = &abLogger{}

type abLogger struct {
	w io.Writer
}

type logfmtEncoder struct {
	*logfmt.Encoder
	buf bytes.Buffer
}

func (l *logfmtEncoder) Reset() {
	l.Encoder.Reset()
	l.buf.Reset()
}

var logfmtEncoderPool = sync.Pool{
	New: func() interface{} {
		var enc logfmtEncoder
		enc.Encoder = logfmt.NewEncoder(&enc.buf)
		return &enc
	},
}

var (
	debugMessage = []byte(color.New(color.FgBlack, color.BgWhite).Sprint("DEBUG"))
	infoMessage  = []byte(color.New(color.FgWhite, color.BgBlue).Sprint("INFO"))
	warnMessage  = []byte(color.New(color.FgWhite, color.BgYellow).Sprint("WARN"))
	errorMessage = []byte(color.New(color.FgWhite, color.BgRed).Sprint("ERROR"))
)

type levelString string

func (s levelString) Format(w io.Writer) {
	switch s {
	case "debug":
		w.Write(debugMessage)
	case "info":
		w.Write(infoMessage)
	case "warn":
		w.Write(warnMessage)
	case "error":
		w.Write(errorMessage)
	default:
		w.Write([]byte(s))
	}
}

func (l *abLogger) Log(keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}

	if len(keyvals)%2 == 1 {
		keyvals = append(keyvals, nil)
	}

	enc := logfmtEncoderPool.Get().(*logfmtEncoder)
	enc.Reset()
	defer logfmtEncoderPool.Put(enc)

	rawKeyvals := make([]interface{}, 0, len(keyvals))

	for i := 0; i < len(keyvals); i += 2 {
		k, v := keyvals[i], keyvals[i+1]
		if k == "level" {
			v = levelString(v.(fmt.Stringer).String())
		}
		if f, ok := v.(ValueFormatter); ok {
			f.Format(&enc.buf)
			enc.buf.Write([]byte(" "))
		} else {
			rawKeyvals = append(rawKeyvals, fix(k), fix(v))
		}
	}

	if len(rawKeyvals) > 0 {
		if eerr := enc.EncodeKeyvals(rawKeyvals...); eerr != nil {
			return eerr
		}
	}

	if eerr := enc.EndRecord(); eerr != nil {
		return eerr
	}

	if _, werr := l.w.Write(enc.buf.Bytes()); werr != nil {
		return werr
	}

	return nil
}

func withLevels(l Logger, options ...level.Option) Logger {
	return level.NewFilter(l, options...)
}

func DefaultProdLogger(options ...level.Option) Logger {
	return NewProdLogger(os.Stdout, options...)
}

func NewProdLogger(w io.Writer, options ...level.Option) Logger {
	return withLevels(&fixerLogger{
		logger: log.NewLogfmtLogger(log.NewSyncWriter(w)),
	}, options...)
}

func DefaultJSONLogger(options ...level.Option) Logger {
	return NewJSONLogger(os.Stdout, options...)
}

func NewJSONLogger(w io.Writer, options ...level.Option) Logger {
	return withLevels(log.NewJSONLogger(log.NewSyncWriter(w)), options...)
}

func DefaultDevLogger(options ...level.Option) Logger {
	return NewDevLogger(os.Stdout, options...)
}

func NewDevLogger(w io.Writer, options ...level.Option) Logger {
	return withLevels(&abLogger{
		w: log.NewSyncWriter(w),
	}, options...)
}

type ValueFormatter interface {
	Format(w io.Writer)
}

// fixerLogger converts all compound keyvals into strings.
type fixerLogger struct {
	logger Logger
}

func (v *fixerLogger) Log(keyvals ...interface{}) error {
	for i, v := range keyvals {
		keyvals[i] = fix(v)
	}

	return v.logger.Log(keyvals...)
}

func fix(v interface{}) interface{} {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return fmt.Sprintf("%#v", v)
	case reflect.Struct:
		return fmt.Sprintf("%+v", v)
	}

	return v
}

func NewStdlibAdapter(logger Logger, options ...log.StdlibAdapterOption) io.Writer {
	return log.NewStdlibAdapter(logger, options...)
}

func Error(logger Logger) Logger {
	return level.Error(logger)
}

func Warn(logger Logger) Logger {
	return level.Warn(logger)
}

func Info(logger Logger) Logger {
	return level.Info(logger)
}

func Debug(logger Logger) Logger {
	return level.Debug(logger)
}

func AllowDebug() Option {
	return level.AllowDebug()
}

func AllowInfo() Option {
	return level.AllowInfo()
}

func AllowWarn() Option {
	return level.AllowWarn()
}

func AllowError() Option {
	return level.AllowError()
}
