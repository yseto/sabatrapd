package oid

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Parse(s string) ([]int, error) {
	var r []int
	for j, str := range strings.Split(s, ".") {
		if str == "" && j == 0 {
			continue
		}
		if str == "" {
			err := fmt.Errorf("invalid %q, remove a empty characters", s)
			return nil, err
		}
		v, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}
		r = append(r, v)
	}
	return r, nil
}

func HasPrefix(s, prefix []int) bool {
	return len(s) >= len(prefix) && reflect.DeepEqual(s[:len(prefix)], prefix)
}
