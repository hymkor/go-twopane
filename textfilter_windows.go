package twopane

import (
	"fmt"
	"os"
	"strings"
)

var surrogatePairOk = os.Getenv("WT_SESSION") != "" && os.Getenv("WT_PROFILE_ID") != ""

func textfilter(s string) string {
	var buffer strings.Builder
	for _, c := range s {
		if !surrogatePairOk && c > 0xFFFF {
			fmt.Fprintf(&buffer, "&#x%X;", c)
		} else {
			buffer.WriteRune(c)
		}
	}
	return buffer.String()
}
