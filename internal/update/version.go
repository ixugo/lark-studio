package update

import (
	"cmp"
	"fmt"
	"strings"
)

// CompareVersions 按数字比较三段版本号，不限制每一段的位数。
func CompareVersions(a, b string) (int, error) {
	aa, err := versionParts(a)
	if err != nil {
		return 0, err
	}
	bb, err := versionParts(b)
	if err != nil {
		return 0, err
	}
	for i := range 3 {
		if n := cmp.Compare(len(aa[i]), len(bb[i])); n != 0 {
			return n, nil
		}
		if n := strings.Compare(aa[i], bb[i]); n != 0 {
			return n, nil
		}
	}
	return 0, nil
}

func versionParts(v string) ([]string, error) {
	if len(v) > 256 {
		return nil, fmt.Errorf("版本号过长")
	}
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("版本号 %q 必须包含三段非负整数", v)
	}
	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("版本号 %q 包含空段", v)
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("版本号 %q 包含非数字", v)
			}
		}
		parts[i] = strings.TrimLeft(part, "0")
	}
	return parts, nil
}

func ShouldPrompt(current, latest, ignored string, manual bool) (bool, error) {
	n, err := CompareVersions(latest, current)
	if err != nil || n <= 0 {
		return false, err
	}
	if !manual && ignored != "" {
		n, err := CompareVersions(latest, ignored)
		if err != nil {
			return false, err
		}
		if n <= 0 {
			return false, nil
		}
	}
	return true, nil
}
