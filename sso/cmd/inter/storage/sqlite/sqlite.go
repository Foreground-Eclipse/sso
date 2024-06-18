package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sso/sso/cmd/inter/domain/models"
	"sso/sso/cmd/inter/storage"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

// New creates a new instance of sqlite storage
func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", op, err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, passHash []byte, email string, dateOfBirth string, fullName string, phoneNumber string, telegramName string) (int64, error) {
	const op = "storage.sqlite.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO Users (passwordHash, email, dateofbirth, fullName, phoneNumber, telegramname)  VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, passHash, email, dateOfBirth, fullName, phoneNumber, telegramName)
	if err != nil {
		var sqliteErr sqlite3.Error

		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// User returns user by email

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.User"
	stmt, err := s.db.Prepare("SELECT id, passwordHash, email, dateofbirth, fullname, phonenumber, telegramname from users where email = ?")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, email)

	var user models.User
	err = row.Scan(&user.ID, &user.PassHash, &user.Email, &user.DateOfBirth, &user.FullName, &user.PhoneNumber, &user.TelegramName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	return user, nil
}

func (s *Storage) App(ctx context.Context, id int) (models.App, error) {
	const op = "storage.sqlite.App"

	stmt, err := s.db.Prepare("SELECT id, name, secret FROM apps WHERE id = ?")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, id)

	var app models.App
	err = row.Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "storage.sqlite.IsAdmin"

	stmt, err := s.db.Prepare("SELECT is_admin FROM users WHERE id = ?")
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, userID)

	var isAdmin bool

	err = row.Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (s *Storage) IsCodeSent(ctx context.Context, userid int) (bool, error) {
	stmt, err := s.db.Prepare("SELECT ConfirmationToken from AccountConfirmations where userid = ?")
	if err != nil {
		return false, nil
	}

	var confirmationToken string
	err = stmt.QueryRowContext(ctx, userid).Scan(&confirmationToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
	}
	return true, nil
}

func (s *Storage) VerifyConfirmationCode(ctx context.Context, userid int, code string) (bool, error) {
	const op = "storage.VerifyConfirmationCode"
	stmt, err := s.db.Prepare("SELECT ConfirmationToken FROM AccountConfirmations WHERE userid = ?")
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var confirmationToken string
	err = stmt.QueryRowContext(ctx, userid).Scan(&confirmationToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrTokenNotFound)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if confirmationToken == code {
		return true, nil
	} else {
		return false, nil
	}
}

func (s *Storage) SaveEmailCode(ctx context.Context, userid int64, confirmationToken string) (int64, error) {
	const op = "storage.sqlite.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO AccountConfirmations(userid, confirmationToken, isconfirmed)  VALUES (?, ?, FALSE)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, userid, confirmationToken)
	if err != nil {
		var sqliteErr sqlite3.Error

		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) SaveEmailConfirmation(ctx context.Context, userID int, confirmationToken string) error {
	const op = "storage.SaveEmailConfirmation"

	// First, retrieve the existing token from the database
	stmt, err := s.db.Prepare("SELECT ConfirmationToken FROM AccountConfirmation WHERE userid = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var existingToken string
	err = stmt.QueryRowContext(ctx, userID).Scan(&existingToken)
	if err != nil {
		// Handle the case where the token does not exist
		return fmt.Errorf("%s: %w", op, err)
	}

	// Compare the existing token with the provided token
	if existingToken == confirmationToken {
		// If they match, update the account status
		return s.UpdateAccountStatus(ctx, int64(userID), confirmationToken)
	} else {
		// If they do not match, return an appropriate error
		return fmt.Errorf("%s: tokens do not match", op)
	}
}

func (s *Storage) UpdateAccountStatus(ctx context.Context, userid int64, token string) error {
	const op = "storage.sqlite.UpdateAccountStatus"

	// Update the account confirmation status to true
	stmt, err := s.db.Prepare("UPDATE AccountConfirmation SET isconfirmed = True WHERE userid = ? AND ConfirmationToken = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, userid, token)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Check if the update was successful
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		// No rows were updated, which means the token did not match
		return fmt.Errorf("%s: no rows updated, token may not match", op)
	}

	return nil
}

func (s *Storage) ConfirmAccount(ctx context.Context, email, token string) error {
	const op = "storage.sqlite.ConfirmAccount"

	// Begin a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Rollback the transaction in case of any error
	defer tx.Rollback()

	// Get the userID from the Users table
	var userID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM Users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%s: no user found with the provided email", op)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	// Update the AccountConfirmations table
	stmt, err := tx.Prepare("UPDATE AccountConfirmations SET isConfirmed = 1 WHERE userID = ? AND ConfirmationToken = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, userID, token)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Check if the update was successful
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		// No rows were updated, which means the token did not match
		return fmt.Errorf("%s: no rows updated, token may not match", op)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) ConfirmAccountTG(ctx context.Context, telegramName string) int {
	const op = "storage.sqlite.ConfirmAccount"

	// Begin a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("%s: %v\n", op, err)
		return 0
	}

	// Rollback the transaction in case of any error
	defer tx.Rollback()

	// Get the userID from the Users table
	var userID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM Users WHERE telegramName = ?", telegramName).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("%s: no user found with the provided telegram name\n", op)
			return 0
		}
		fmt.Printf("%s: %v\n", op, err)
		return 0
	}

	// Check if the account is already confirmed
	var isConfirmed bool
	err = tx.QueryRowContext(ctx, "SELECT isConfirmed FROM AccountConfirmations WHERE userID = ?", userID).Scan(&isConfirmed)
	if err != nil {
		fmt.Printf("%s: %v\n", op, err)
		return 0
	}
	if isConfirmed {
		return -1
	}

	// Update the isConfirmed status to true
	result, err := tx.ExecContext(ctx, "UPDATE AccountConfirmations SET isConfirmed = 1 WHERE userID = ?", userID)
	if err != nil {
		fmt.Printf("%s: %v\n", op, err)
		return 0
	}

	// Check if the update was successful
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("%s: %v\n", op, err)
		return 0
	}
	if rowsAffected == 0 {
		fmt.Printf("%s: no rows updated, user may not exist\n", op)
		return 0
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		fmt.Printf("%s: %v\n", op, err)
		return 0
	}

	return 1
}
