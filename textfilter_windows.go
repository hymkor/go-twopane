package twopane

import (
	"fmt"
	"strings"
)

func textfilter(s string) string {
	var buffer strings.Builder
	for _, c := range s {
		if c > 0xFFFF {
			fmt.Fprintf(&buffer, "&#x%X;", c)
		} else {
			buffer.WriteRune(c)
		}
	}
	return buffer.String()
}
