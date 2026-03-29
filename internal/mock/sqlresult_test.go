package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSqlResult_LastInsertId(t *testing.T) {
	t.Run("获取最后插入ID", func(t *testing.T) {
		result := &SqlResult{
			LastID: 42,
			RA:     1,
		}

		id, err := result.LastInsertId()

		assert.NoError(t, err)
		assert.Equal(t, int64(42), id)
	})

	t.Run("ID为0", func(t *testing.T) {
		result := &SqlResult{
			LastID: 0,
			RA:     1,
		}

		id, err := result.LastInsertId()

		assert.NoError(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("ID为负数", func(t *testing.T) {
		result := &SqlResult{
			LastID: -1,
			RA:     1,
		}

		id, err := result.LastInsertId()

		assert.NoError(t, err)
		assert.Equal(t, int64(-1), id)
	})

	t.Run("大ID值", func(t *testing.T) {
		result := &SqlResult{
			LastID: 9223372036854775807, // Max int64
			RA:     1,
		}

		id, err := result.LastInsertId()

		assert.NoError(t, err)
		assert.Equal(t, int64(9223372036854775807), id)
	})
}

func TestSqlResult_RowsAffected(t *testing.T) {
	t.Run("单行受影响", func(t *testing.T) {
		result := &SqlResult{
			LastID: 1,
			RA:     1,
		}

		ra, err := result.RowsAffected()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), ra)
	})

	t.Run("多行受影响", func(t *testing.T) {
		result := &SqlResult{
			LastID: 0,
			RA:     5,
		}

		ra, err := result.RowsAffected()

		assert.NoError(t, err)
		assert.Equal(t, int64(5), ra)
	})

	t.Run("无行受影响", func(t *testing.T) {
		result := &SqlResult{
			LastID: 0,
			RA:     0,
		}

		ra, err := result.RowsAffected()

		assert.NoError(t, err)
		assert.Equal(t, int64(0), ra)
	})

	t.Run("负数行数", func(t *testing.T) {
		result := &SqlResult{
			LastID: 0,
			RA:     -1,
		}

		ra, err := result.RowsAffected()

		assert.NoError(t, err)
		assert.Equal(t, int64(-1), ra)
	})
}

func TestSqlResult_BothMethods(t *testing.T) {
	t.Run("同时调用两个方法", func(t *testing.T) {
		result := &SqlResult{
			LastID: 100,
			RA:     3,
		}

		id, err1 := result.LastInsertId()
		ra, err2 := result.RowsAffected()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, int64(100), id)
		assert.Equal(t, int64(3), ra)
	})
}
