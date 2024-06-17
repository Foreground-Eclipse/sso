package auth

import (
	"context"
	"errors"
	"sso/sso/cmd/inter/services/auth"
	"sso/sso/cmd/inter/storage"

	v1 "github.com/Foreground-Eclipse/testprotos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const emptyValue = 0

type Auth interface {
	Login(ctx context.Context,
		email string,
		password string,
		appID int) (token string, err error)
	RegisterNewUser(ctx context.Context,
		email string,
		password string,
		dateOfBirth string,
		fullName string,
		phoneNumber string,
		telegramName string) (userID int64, err error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
	EmailVerification(ctx context.Context,
		email string,
		verificationCode string) (bool, error)
}

type serverAPI struct {
	v1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	v1.RegisterAuthServer(gRPC, &serverAPI{auth: auth})
}

func (s *serverAPI) Login(ctx context.Context,
	req *v1.LoginRequest) (*v1.LoginResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "Email cant be empty")
	}

	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "Password cant be empty")
	}

	if req.GetAppId() == emptyValue {
		return nil, status.Error(codes.InvalidArgument, "AppId is required")
	}

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), int(req.GetAppId()))

	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &v1.LoginResponse{
		Token: token,
	}, nil
}

func (s *serverAPI) Register(ctx context.Context,
	req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword(), req.GetDateOfBirth(), req.GetFullName(), req.GetPhoneNumber(), req.GetTelegramName())
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Error(codes.Internal, "Internal error")
	}

	return &v1.RegisterResponse{
		UserId: userID,
	}, nil

}

func (s *serverAPI) IsAdmin(ctx context.Context,
	req *v1.IsAdminRequest) (*v1.IsAdminResponse, error) {
	if err := validateIsAdmin(req); err != nil {
		return nil, err
	}

	isAdmin, err := s.auth.IsAdmin(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &v1.IsAdminResponse{
		IsAdmin: isAdmin,
	}, nil
}

func (s *serverAPI) EmailVerification(ctx context.Context,
	req *v1.EmailVerificationRequest) (*v1.EmailVerificationResponse, error) {
	if err := validateEmailVerification(req); err != nil {
		return nil, err
	}
	isVerified, err := s.auth.EmailVerification(ctx, req.GetEmail(), req.GetSecretCode())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &v1.EmailVerificationResponse{
		IsVerified: isVerified,
	}, nil
}

func validateRegister(req *v1.RegisterRequest) error {
	if req.GetDateOfBirth() == "" {
		return status.Error(codes.InvalidArgument, "Date of birth cant be empty")
	}
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "Email cant be empty")
	}
	if req.GetFullName() == "" {
		return status.Error(codes.InvalidArgument, "Full name cant be empty")
	}
	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, "Password cant be empty")
	}
	if req.GetPhoneNumber() == "" {
		return status.Error(codes.InvalidArgument, "Phone number cant be empty")
	}
	if req.GetTelegramName() == "" {
		return status.Error(codes.InvalidArgument, "Telegram name cant be empty")
	}
	return nil
}

func validateIsAdmin(req *v1.IsAdminRequest) error {
	if req.GetUserId() == emptyValue {
		return status.Error(codes.InvalidArgument, "User id cant be empty")
	}
	return nil
}

func validateEmailVerification(req *v1.EmailVerificationRequest) error {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "Email cant be empty")
	}
	if req.GetSecretCode() == "" {
		return status.Error(codes.InvalidArgument, "Verification code cant be empty")
	}
	return nil
}
