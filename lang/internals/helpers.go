package internals

import (
	"sort"
)

type Mode = string

const (
	VIDEO Mode = "video"
	IMAGE Mode = "image"
)

func CheckIfElementExist(slice []string, element string) bool {
	sort.Strings(slice)
	idx := sort.SearchStrings(slice, element)
	return idx < len(slice) && slice[idx] == element
}
