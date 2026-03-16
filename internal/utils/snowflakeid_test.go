package utils

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitSonyflake(t *testing.T) {
	tests := []struct {
		name      string
		machineID uint16
		startTime string
		wantErr   bool
	}{
		{
			name:      "正常初始化",
			machineID: 1,
			startTime: "2024-01-01",
			wantErr:   false,
		},
		{
			name:      "不同机器ID",
			machineID: 100,
			startTime: "2024-01-01",
			wantErr:   false,
		},
		{
			name:      "无效的时间格式",
			machineID: 1,
			startTime: "invalid-time",
			wantErr:   true,
		},
		{
			name:      "空的开始时间",
			machineID: 1,
			startTime: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每次测试前重置 flake
			flake = nil

			err := InitSonyflake(tt.machineID, tt.startTime)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, flake)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	t.Run("未初始化时生成ID", func(t *testing.T) {
		// 重置 flake
		flake = nil

		_, err := GenerateID()
		assert.Error(t, err)
	})

	t.Run("正常生成ID", func(t *testing.T) {
		// 初始化
		err := InitSonyflake(1, "2024-01-01")
		require.NoError(t, err)

		id, err := GenerateID()
		assert.NoError(t, err)
		assert.Greater(t, id, int64(0))
	})

	t.Run("生成的ID唯一性", func(t *testing.T) {
		// 初始化
		err := InitSonyflake(1, "2024-01-01")
		require.NoError(t, err)

		// 生成多个ID
		idCount := 1000
		ids := make(map[int64]bool)

		for i := 0; i < idCount; i++ {
			id, err := GenerateID()
			require.NoError(t, err)

			// 检查ID是否重复
			assert.False(t, ids[id], "ID应该唯一，但发现重复: %d", id)
			ids[id] = true
		}

		assert.Equal(t, idCount, len(ids), "生成的ID数量应该等于请求数量")
	})

	t.Run("并发生成ID唯一性", func(t *testing.T) {
		// 初始化
		err := InitSonyflake(1, "2024-01-01")
		require.NoError(t, err)

		var wg sync.WaitGroup
		idChan := make(chan int64, 10000)
		goroutineCount := 10
		idsPerGoroutine := 100

		for i := 0; i < goroutineCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id, err := GenerateID()
					if err != nil {
						t.Errorf("生成ID失败: %v", err)
						return
					}
					idChan <- id
				}
			}()
		}

		// 等待所有goroutine完成
		go func() {
			wg.Wait()
			close(idChan)
		}()

		// 收集所有ID并检查唯一性
		ids := make(map[int64]bool)
		for id := range idChan {
			assert.False(t, ids[id], "并发生成的ID应该唯一，但发现重复: %d", id)
			ids[id] = true
		}

		expectedCount := goroutineCount * idsPerGoroutine
		assert.Equal(t, expectedCount, len(ids), "生成的ID数量应该正确")
	})
}

func TestGenerateID_Order(t *testing.T) {
	// 初始化
	err := InitSonyflake(1, "2024-01-01")
	require.NoError(t, err)

	// 生成多个ID，检查大致递增趋势（不严格保证连续递增）
	prevID, err := GenerateID()
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		require.NoError(t, err)

		// 新ID应该大于等于前一个ID（在正常情况下是递增的）
		assert.GreaterOrEqual(t, id, prevID, "ID应该是递增的")
		prevID = id
	}
}

func TestGenerateID_DifferentMachineIDs(t *testing.T) {
	// 测试不同机器ID生成的ID不会冲突
	machineIDs := []uint16{1, 2, 3}
	allIDs := make(map[int64]uint16)

	for _, machineID := range machineIDs {
		// 重置并初始化
		flake = nil
		err := InitSonyflake(machineID, "2024-01-01")
		require.NoError(t, err)

		// 生成一些ID
		for i := 0; i < 10; i++ {
			id, err := GenerateID()
			require.NoError(t, err)

			// 检查是否与其他机器的ID冲突
			if existingMachine, exists := allIDs[id]; exists {
				t.Errorf("不同机器生成的ID冲突: %d 由机器 %d 和 %d 都生成了", id, existingMachine, machineID)
			}
			allIDs[id] = machineID
		}
	}
}

func BenchmarkGenerateID(b *testing.B) {
	// 初始化
	err := InitSonyflake(1, "2024-01-01")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateID()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateID_Parallel(b *testing.B) {
	// 初始化
	err := InitSonyflake(1, "2024-01-01")
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := GenerateID()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestSonyflake_TimeBased 测试基于时间的特性
func TestSonyflake_TimeBased(t *testing.T) {
	// 使用过去的时间初始化
	err := InitSonyflake(1, "2020-01-01")
	require.NoError(t, err)

	id1, err := GenerateID()
	require.NoError(t, err)

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	id2, err := GenerateID()
	require.NoError(t, err)

	// 由于时间推移，ID应该不同
	assert.NotEqual(t, id1, id2, "不同时间生成的ID应该不同")
}
