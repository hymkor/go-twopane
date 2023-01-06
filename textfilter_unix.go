//go:build !windows
// +build !windows

package twopane

func textfilter(s string) string {
	return s
}
