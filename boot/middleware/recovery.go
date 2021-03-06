package middleware

import (
	"api/boot/http"
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http/httputil"
	"runtime"
)

var (
	dunno		= []byte("???")
	centerDot	= []byte("·")
	dot			= []byte(".")
	slash		= []byte("/")
)

func Recovery() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				request, _ := httputil.DumpRequest(ctx.Request, false)
				stack := runtimeStack(3)
				//TODO: log
				//ctx.DealSucc = false
				fmt.Printf("[Recovery] panic recovered:\n%s\n%s\n%s\n%s\n",
					string(request),
					ctx.GetString(RawBodyKey),
					err,
					stack,
				)
				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		ctx.Next()
	}
}

// stack returns a nicely formatted stack frame, skipping skip frames
func runtimeStack(skip int) []byte {
	// returned data
	buf := new(bytes.Buffer)
	// As we loop, we open files and read them. These variables record the currently loaded file
	var lines [][]byte
	var lastFile string

	// Skip the expected number of frames
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// Print this much at least. If we can't find the source, it won't show.
		_, _ = fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		_, _ = fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included. Plus, it has center dots
	// That is, we see
	// runtime/debug.*T·ptrMethod
	// and want
	// *T.ptrMethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash + 1 : ]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period + 1 : ]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}
