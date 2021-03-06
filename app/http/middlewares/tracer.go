package middlewares

import (
	"fmt"
	"strconv"
	"time"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	v3 "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"github.com/gin-gonic/gin"
	"github.com/qit-team/snow-core/log/logger"
	"github.com/qit-team/snow/app/http/trace"
)

const (
	componentIDGOHttpServer = 5004
)

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer, err := trace.Tracer()
		if err != nil {
			logger.Error(c, "Trace", err.Error())
			c.Next()
			return
		}
		r := c.Request
		operationName := fmt.Sprintf("/%s%s", r.Method, r.URL.Path)
		span, ctx, err := tracer.CreateEntrySpan(c, operationName, func() (string, error) {
			// 从http头部捞取上一层的调用链信息, 当前使用v3版本的协议
			// https://github.com/apache/skywalking/blob/master/docs/en/protocols/Skywalking-Cross-Process-Propagation-Headers-Protocol-v3.md
			return r.Header.Get(propagation.Header), nil
		})
		if err != nil {
			logger.Error(c, "Trace", err.Error())
			c.Next()
			return
		}
		span.SetComponent(componentIDGOHttpServer)
		// 可以自定义tag
		span.Tag(go2sky.TagHTTPMethod, r.Method)
		span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", r.Host, r.URL.Path))
		span.SetSpanLayer(v3.SpanLayer_Http)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		code := c.Writer.Status()
		if code >= 400 {
			span.Error(time.Now(), fmt.Sprintf("Error on handling request, statusCode: %d", code))
		}
		span.Tag(go2sky.TagStatusCode, strconv.Itoa(code))
		span.End()
	}
}
