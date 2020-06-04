package stackdriver

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logKeyServiceContext        = "serviceContext"
	logKeyContextHTTPRequest    = "context.httpRequest"
	logKeyContextUser           = "context.user"
	logKeyContextReportLocation = "context.reportLocation"
)

var logLevelSeverity = map[zapcore.Level]string{
	zapcore.DebugLevel:  "DEBUG",
	zapcore.InfoLevel:   "INFO",
	zapcore.WarnLevel:   "WARNING",
	zapcore.ErrorLevel:  "ERROR",
	zapcore.DPanicLevel: "CRITICAL",
	zapcore.PanicLevel:  "ALERT",
	zapcore.FatalLevel:  "EMERGENCY",
}

var EncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "timestamp",
	LevelKey:       "severity",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "message",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    EncodeLevel,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.MillisDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

type Core struct {
	zapcore.Core

	SetReportLocation bool

	ctx *Context
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	fields, ctx := c.extractCtx(fields)

	return &Core{
		Core:              c.Core.With(fields),
		SetReportLocation: c.SetReportLocation,
		ctx:               ctx,
	}
}

func (c *Core) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}

	return ce
}

func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	loc := c.getReportLocationFromEntry(entry)

	if loc != nil {
		fields = append(fields, LogReportLocation(loc))
	}

	fields, ctx := c.extractCtx(fields)
	fields = append(fields, zap.Object("context", ctx))

	entry.Message = c.appendFields(entry.Message, fields)

	return c.Core.Write(entry, fields)
}

func (c *Core) Sync() error {
	return c.Core.Sync()
}

func (c *Core) appendFields(str string, fields []zapcore.Field) string {
	builder := strings.Builder{}
	builder.WriteString(str)
	for _, field := range fields {
		if field.Key == "context" {
			continue
		}
		builder.WriteString(" ")
		builder.WriteString(field.Key)
		builder.WriteString("=")
		builder.WriteString(fmt.Sprintf("%v", c.fieldValueToString(field)))
	}

	return builder.String()
}

func (c *Core) fieldValueToString(field zapcore.Field) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered fieldValueToString:", r)
		}
	}()

	switch field.Type {
	case zapcore.ArrayMarshalerType:
		return ""
	case zapcore.ObjectMarshalerType:
		return ""
	case zapcore.BinaryType:
		return ""
	case zapcore.BoolType:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.ByteStringType:
		return ""
	case zapcore.Complex128Type:
		return ""
	case zapcore.Complex64Type:
		return ""
	case zapcore.DurationType:
		return strconv.FormatInt(field.Integer / 1000000, 10)
	case zapcore.Float64Type:
		return strconv.FormatFloat(math.Float64frombits(uint64(field.Integer)), 'e', 2, 64)
	case zapcore.Float32Type:
		return strconv.FormatFloat(float64(math.Float32frombits(uint32(field.Integer))), 'e', 2, 32)
	case zapcore.Int64Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Int32Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Int16Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Int8Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.StringType:
		return field.String
	case zapcore.TimeType:
		return time.Unix(0, field.Integer).String()
	case zapcore.TimeFullType:
		return field.Interface.(time.Time).String()
	case zapcore.Uint64Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Uint32Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Uint16Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Uint8Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.UintptrType:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.ReflectType:
		return ""
	case zapcore.NamespaceType:
		return ""
	case zapcore.StringerType:
		return field.Interface.(fmt.Stringer).String()
	case zapcore.ErrorType:
		return field.Interface.(error).Error()
	case zapcore.SkipType:
		return ""
	}
	return ""
}

func (c *Core) extractCtx(fields []zapcore.Field) ([]zapcore.Field, *Context) {
	output := []zapcore.Field{}
	ctx := c.cloneCtx()

	for _, f := range fields {
		switch f.Key {
		case logKeyContextHTTPRequest:
			ctx.HTTPRequest = f.Interface.(*HTTPRequest)
		case logKeyContextReportLocation:
			ctx.ReportLocation = f.Interface.(*ReportLocation)
		case logKeyContextUser:
			ctx.User = f.String
		default:
			output = append(output, f)
		}
	}

	return output, ctx
}

func (c *Core) cloneCtx() *Context {
	if c.ctx == nil {
		return &Context{}
	}

	return c.ctx.Clone()
}

func (c *Core) getReportLocationFromEntry(entry zapcore.Entry) *ReportLocation {
	if !c.SetReportLocation {
		return nil
	}

	caller := entry.Caller

	if !caller.Defined {
		return nil
	}

	loc := &ReportLocation{
		FilePath:   caller.File,
		LineNumber: caller.Line,
	}

	if fn := runtime.FuncForPC(caller.PC); fn != nil {
		loc.FunctionName = fn.Name()
	}

	return loc
}

func LogServiceContext(ctx *ServiceContext) zapcore.Field {
	return zap.Object(logKeyServiceContext, ctx)
}

func LogHTTPRequest(req *HTTPRequest) zapcore.Field {
	return zap.Object(logKeyContextHTTPRequest, req)
}

func LogUser(user string) zapcore.Field {
	return zap.String(logKeyContextUser, user)
}

func LogReportLocation(loc *ReportLocation) zapcore.Field {
	return zap.Object(logKeyContextReportLocation, loc)
}

func EncodeLevel(lv zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(logLevelSeverity[lv])
}
