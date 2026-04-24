// Package handlers contains gRPC server handler implementations.
package handlers

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yourorg/microservice-platform/backend/internal/domain/user"
	pkgerrors "github.com/yourorg/microservice-platform/backend/pkg/errors"
	pb "github.com/yourorg/microservice-platform/backend/internal/grpc/pb/user/v1"
)

// UserHandler implements pb.UserServiceServer.
type UserHandler struct {
	pb.UnimplementedUserServiceServer
	svc    user.Service
	logger *zap.Logger
}

func NewUserHandler(svc user.Service, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		svc:    svc,
		logger: logger.Named("grpc-user-handler"),
	}
}

// ============================================================
// Auth RPCs
// ============================================================

func (h *UserHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	u, accessToken, refreshToken, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
		User:         toProtoUser(u),
	}, nil
}

func (h *UserHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	accessToken, err := h.svc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   900,
	}, nil
}

// ============================================================
// CRUD RPCs
// ============================================================

func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	cmd := user.CreateCommand{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      user.RoleMember,
	}

	u, err := h.svc.Create(ctx, cmd)
	if err != nil {
		return nil, toGRPCError(err)
	}

	h.logger.Info("grpc: user created", zap.String("id", u.ID.String()))
	return &pb.CreateUserResponse{User: toProtoUser(u)}, nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %s", req.Id)
	}

	u, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetUserResponse{User: toProtoUser(u)}, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %s", req.Id)
	}

	cmd := user.UpdateCommand{
		ID:        id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	u, err := h.svc.Update(ctx, cmd)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UpdateUserResponse{User: toProtoUser(u)}, nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %s", req.Id)
	}

	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *UserHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	filter := user.ListFilter{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		Order:    req.Order,
	}

	users, total, err := h.svc.List(ctx, filter)
	if err != nil {
		return nil, toGRPCError(err)
	}

	protoUsers := make([]*pb.User, len(users))
	for i, u := range users {
		protoUsers[i] = toProtoUser(u)
	}

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	pages := int32((total + int64(pageSize) - 1) / int64(pageSize))

	return &pb.ListUsersResponse{
		Users: protoUsers,
		Total: total,
		Page:  req.Page,
		Pages: pages,
	}, nil
}

// ============================================================
// Mappers
// ============================================================

func toProtoUser(u *user.User) *pb.User {
	pu := &pb.User{
		Id:        u.ID.String(),
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
	return pu
}

// toGRPCError maps domain/application errors to gRPC status codes.
func toGRPCError(err error) error {
	switch {
	case pkgerrors.IsNotFound(err):
		return status.Errorf(codes.NotFound, err.Error())
	case pkgerrors.IsAlreadyExists(err):
		return status.Errorf(codes.AlreadyExists, err.Error())
	case pkgerrors.IsUnauthenticated(err):
		return status.Errorf(codes.Unauthenticated, err.Error())
	case pkgerrors.IsPermissionDenied(err):
		return status.Errorf(codes.PermissionDenied, err.Error())
	case pkgerrors.IsValidation(err):
		return status.Errorf(codes.InvalidArgument, err.Error())
	case err == user.ErrInvalidCredentials:
		return status.Errorf(codes.Unauthenticated, "invalid credentials")
	case err == user.ErrInactive:
		return status.Errorf(codes.PermissionDenied, "account is inactive")
	case err == user.ErrAlreadyExists:
		return status.Errorf(codes.AlreadyExists, "user already exists")
	case err == user.ErrNotFound:
		return status.Errorf(codes.NotFound, "user not found")
	default:
		return status.Errorf(codes.Internal, "internal server error")
	}
}
