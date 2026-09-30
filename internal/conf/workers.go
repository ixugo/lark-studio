package conf

import "fmt"

// MaxTaskWorkers 与设置页面的并发任务数范围一致。
const MaxTaskWorkers = 8

func ValidateWorkers(n int) error {
	if n < 1 || n > MaxTaskWorkers {
		return fmt.Errorf("并发任务数必须为 1~%d 的整数", MaxTaskWorkers)
	}
	return nil
}
