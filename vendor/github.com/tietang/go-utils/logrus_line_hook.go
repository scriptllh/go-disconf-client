package utils

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "runtime"
    "strings"
)

func NewLogLineNumHook() *LogLineNumHook {
    return &LogLineNumHook{
        EnableFileLogLine: true,
        EnableLogFuncName: true,
        Skip:              5,
    }
}

type LogLineNumHook struct {
    EnableFileLogLine bool
    EnableLogFuncName bool
    Skip              int
}

func (hooks LogLineNumHook) Levels() []log.Level {
    return log.AllLevels
}

func (hook *LogLineNumHook) Fire(entry *log.Entry) error {

    file, function, line := hook.findCaller(hook.Skip)

    if hook.EnableFileLogLine && hook.EnableLogFuncName {
        entry.Message = fmt.Sprintf("[%s(%s:%d)] [%s]", function, file, line, entry.Message)
    }
    //router/route_table.go(43)
    if hook.EnableFileLogLine && !hook.EnableLogFuncName {
        entry.Message = fmt.Sprintf("[%s(%d)] %s", file, line, entry.Message)
    }
    //microservice-gateway/v1/router.(*RouteTable).AddRoutePattern(43)
    if !hook.EnableFileLogLine && hook.EnableLogFuncName {
        entry.Message = fmt.Sprintf("[%s(%d)] %s", function, line, entry.Message)
    }

    return nil
}

func (hook *LogLineNumHook) findCaller(skip int) (string, string, int) {
    var (
        pc       uintptr
        file     string
        function string
        line     int
    )
    for i := 0; i < 10; i++ {
        pc, file, line = hook.getCaller(skip + i)
        if !strings.HasPrefix(file, "logrus/") {
            break
        }
    }
    if pc != 0 && hook.EnableLogFuncName {
        frames := runtime.CallersFrames([]uintptr{pc})
        frame, _ := frames.Next()
        function = frame.Function
    }

    return file, function, line
}

func (hook *LogLineNumHook) getCaller(skip int) (uintptr, string, int) {
    pc, file, line, ok := runtime.Caller(skip)
    if !ok {
        return 0, "", 0
    }

    n := 0
    for i := len(file) - 1; i > 0; i-- {
        if file[i] == '/' {
            n += 1
            if n >= 2 {
                file = file[i+1:]
                break
            }
        }
    }

    return pc, file, line
}
