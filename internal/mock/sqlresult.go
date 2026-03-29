package mock

// SqlResult 模拟 sql.Result
type SqlResult struct {
	LastID int64
	RA     int64
}

func (m *SqlResult) LastInsertId() (int64, error) {
	return m.LastID, nil
}

func (m *SqlResult) RowsAffected() (int64, error) {
	return m.RA, nil
}
