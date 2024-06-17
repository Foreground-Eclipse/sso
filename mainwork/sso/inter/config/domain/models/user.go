package models

type User struct {
	ID           int64
	Email        string
	FullName     string
	DateOfBirth  string
	PhoneNumber  string
	TelegramName string
	PassHash     []byte
}
