package proxy

import (
	"fmt"
)

type ProxyError struct {
	StatusCode   int
	ContentType  string
	ResponseBody []byte
}

func (e *ProxyError) Error() string {
	return fmt.Sprintf("unexpected status code %d", e.StatusCode)
}
