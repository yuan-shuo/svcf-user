package utils

import (
	"errors"
	"time"

	"github.com/sony/sonyflake"
)

var flake *sonyflake.Sonyflake

// InitSonyflake 初始化索尼雪花
func InitSonyflake(machineID uint16, startTime string) error {
	st, err := time.Parse("2006-01-02", startTime)
	if err != nil {
		return err
	}

	settings := sonyflake.Settings{
		StartTime: st,
		MachineID: func() (uint16, error) {
			return machineID, nil
		},
	}

	flake = sonyflake.NewSonyflake(settings)
	if flake == nil {
		return errors.New("failed to create sonyflake instance")
	}
	return nil
}

// GenerateID 生成雪花ID
func GenerateID() (int64, error) {
	if flake == nil {
		return 0, errors.New("sonyflake not initialized")
	}
	id, err := flake.NextID()
	return int64(id), err // 最终将id转换为int64
}
