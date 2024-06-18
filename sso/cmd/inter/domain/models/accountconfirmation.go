package models

type AccountConfirmation struct {
	ConfirmationID    int
	UserID            int
	ConfirmationToken string
	IsConfirmed       bool
}
