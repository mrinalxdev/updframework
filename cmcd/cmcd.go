package cmcd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderRequest = "CMCD-Request"
	HeaderObject  = "CMCD-Object"
	HeaderStatus  = "CMCD-Status"
	HeaderSession = "CMCD-Session"
)

type Info struct {
	Request
	Object
	Status
	Sesssion

	Custom map[string]any
}

func (info Info) Encode() string {

	ss := make([]string, 4)
	ss[0] = info.Request.Encode()
	ss[1] = info.Object.Encode()
	ss[2] = info.Status.Encode()
	ss[3] = info.Sesssion.Encode()

	if info.Custom != nil && len(info.Custom) > 0 {
		for k, v := range info.Custom {
			switch v.(type) {
			case string:
				ss = append(ss, fmt.Sprintf("%s=%q", k, v))
			case int:
				ss = append(ss, fmt.Sprintf("%s=%d", k, v))
			case bool:
				ss = append(ss, k)
			default:
				ss = append(ss, fmt.Sprintf("%s=%q", k, v))
			}
		}
	}
	var noEmpty []string
	for _, s := range ss {
		if s == "" {
			continue
		}

		noEmpty = append(noEmpty, s)
	}

	return strings.Join(noEmpty, ",")
}
