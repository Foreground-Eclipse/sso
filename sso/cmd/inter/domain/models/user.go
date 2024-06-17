package models

type User struct {
	ID           int64
	Email        string
	PassHash     []byte
	TelegramName string
	DateOfBirth  string
	FullName     string
	PhoneNumber  string
}
