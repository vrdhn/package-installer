package recipe

import (
	"regexp"
	"strings"
)

// CompileAnchored compiles a regex with ^...$ anchors if not already present.
func CompileAnchored(pattern string) (*regexp.Regexp, error) {
	anchored := pattern
	if !strings.HasPrefix(anchored, "^") {
		anchored = "^" + anchored
	}
	if !strings.HasSuffix(anchored, "$") {
		anchored = anchored + "$"
	}
	return regexp.Compile(anchored)
}
