package twopane

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/windows"
)

func atou(bin []byte) (string, error) {
	acp := windows.GetACP()
	length, err := windows.MultiByteToWideChar(acp, 0, &bin[0], int32(len(bin)), nil, 0)
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, length)
	_, err = windows.MultiByteToWideChar(acp, 0, &bin[0], int32(len(bin)), &buffer[0], length)
	if err != nil {
		return "", err
	}
	return windows.UTF16ToString(buffer), nil
}

func textfilter(s string) string {
	if !utf8.ValidString(s) {
		if t, err := atou([]byte(s)); err == nil {
			s = t
		}
	}
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
