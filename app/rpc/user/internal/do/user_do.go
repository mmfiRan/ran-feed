package do

import "time"

// UserDO 用户领域对象
type UserDO struct {
	ID           int64
	Username     string
	Nickname     string
	Avatar       string
	Bio          string
	Mobile       string
	Email        string
	PasswordHash string
	PasswordSalt string
	Gender       int32
	Birthday     *time.Time
	Status       int32
	CreatedBy    int64
	UpdatedBy    int64
}
