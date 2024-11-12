package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sso/sso/cmd/inter/domain/models"
	"sso/sso/cmd/inter/jwt"
	"sso/sso/cmd/inter/storage"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAppId       = errors.New("invalid app id")
	ErrUserExists         = errors.New("user already exists")
)

type Auth struct {
	log          *slog.Logger
	usrSaver     UserSaver
	usrProvider  UserProvider
	appProvider  AppProvider
	emailSaver   EmailConfirmationSaver
	emailUpdater EmailUpdater
	tokenTTL     time.Duration
}

// EmailVerification implements auth.Auth.

type UserSaver interface {
	SaveUser(ctx context.Context,
		passHash []byte,
		email string,
		dateOfBirth string,
		fullName string,
		phoneNumber string,
		telegramName string,
	) (uid int64, err error)
}

type EmailConfirmationSaver interface {
	SaveEmailCode(ctx context.Context, userID int64, confirmationToken string) (int64, error)
}
type EmailUpdater interface {
	ConfirmAccount(ctx context.Context, email string, confirmationToken string) error
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
	IsCodeSent(ctx context.Context, userID int) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

// New returns a new instance of the auth service
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	emailSaver EmailConfirmationSaver,
	emailUpdater EmailUpdater,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		usrSaver:     userSaver,
		usrProvider:  userProvider,
		emailSaver:   emailSaver,
		log:          log,
		appProvider:  appProvider,
		emailUpdater: emailUpdater,
		tokenTTL:     tokenTTL,
	}
}

func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int,
) (string, error) {
	const op = "auth.Login"

	log := a.log.With(slog.String("op", op), slog.String("username", email))

	a.log.Info("Attempting to login the user", email)

	user, err := a.usrProvider.User(ctx, email)
	fmt.Print(user)
	fmt.Print(user.PassHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", err)

			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		a.log.Error("failed to get the user", err)

		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", err)

		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("logged successfully")

	token, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate the token", err)

		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (a *Auth) SendConfirmationCode(ctx context.Context, receiptorEmail string, userid int64) (bool, error) {
	const op = "services.auth.SendConfirmationCode"

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	randomNumber := r.Intn(90000) + 10000
	m := gomail.NewMessage()
	m.SetHeader("From", "username@gmail.com")
	m.SetHeader("To", receiptorEmail)
	m.SetHeader("Subject", "Confirmation email")
	confirmationString := fmt.Sprintf("Hello, your confirmation code is %d", randomNumber)
	m.SetBody("text/html", confirmationString)

	d := gomail.NewDialer("smtp.gmail.com", 587, "username@gmail.com", "")

	if err := d.DialAndSend(m); err != nil {
		return false, err
	}
	_, err := a.emailSaver.SaveEmailCode(ctx, userid, strconv.Itoa(randomNumber))
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil

}

func (a *Auth) EmailVerification(ctx context.Context, email string, verificationCode string) (bool, error) {
	const op = "services.auth.EmailVerification"

	err := a.emailUpdater.ConfirmAccount(ctx, email, verificationCode)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (a *Auth) RegisterNewUser(
	ctx context.Context,
	email string,
	password string,
	dateOfBirth string,
	fullName string,
	phoneNumber string,
	telegramName string,
) (int64, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		log.Error("failed to generate password hash", err)
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.usrSaver.SaveUser(ctx, passHash, email, dateOfBirth, fullName, phoneNumber, telegramName)
	if err != nil {
		log.Error("Failed to save the user", err)

		return 0, fmt.Errorf("%s: %w", op, err)
	}
	a.SendConfirmationCode(ctx, email, id)

	return id, nil
}

func (a *Auth) IsAdmin(
	ctx context.Context,
	userID int64,
) (bool, error) {
	const op = "auth.IsAdmin"
	isAdmin, err := a.usrProvider.IsAdmin(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)

	}
	return isAdmin, nil
}

// TODO:  sdelat ostalnie ruchki)
func (a *Auth) VerifyEmail(
	ctx context.Context,
	email string,
	code string,
) (bool, error) {

	const op = "test"
	err := a.emailUpdater.ConfirmAccount(ctx, email, code)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil

}
