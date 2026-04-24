package handlers

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yourorg/microservice-platform/backend/internal/domain/product"
	pkgerrors "github.com/yourorg/microservice-platform/backend/pkg/errors"
	pb "github.com/yourorg/microservice-platform/backend/internal/grpc/pb/product/v1"
)

// ProductHandler implements pb.ProductServiceServer.
type ProductHandler struct {
	pb.UnimplementedProductServiceServer
	svc    product.Service
	logger *zap.Logger
}

func NewProductHandler(svc product.Service, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{svc: svc, logger: logger.Named("grpc-product-handler")}
}

func (h *ProductHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	p, err := h.svc.Create(ctx, product.CreateCommand{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	})
	if err != nil {
		return nil, toProductGRPCError(err)
	}
	return &pb.CreateProductResponse{Product: toProtoProduct(p)}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id")
	}
	p, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return nil, toProductGRPCError(err)
	}
	return &pb.GetProductResponse{Product: toProtoProduct(p)}, nil
}

func (h *ProductHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id")
	}
	p, err := h.svc.Update(ctx, product.UpdateCommand{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	})
	if err != nil {
		return nil, toProductGRPCError(err)
	}
	return &pb.UpdateProductResponse{Product: toProtoProduct(p)}, nil
}

func (h *ProductHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id")
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toProductGRPCError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *ProductHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	filter := product.ListFilter{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		Category: req.Category,
		Search:   req.Search,
		MinPrice: req.MinPrice,
		MaxPrice: req.MaxPrice,
		SortBy:   req.SortBy,
		Order:    req.Order,
	}
	products, total, err := h.svc.List(ctx, filter)
	if err != nil {
		return nil, toProductGRPCError(err)
	}

	proto := make([]*pb.Product, len(products))
	for i, p := range products {
		proto[i] = toProtoProduct(p)
	}

	pageSize := req.PageSize
	if pageSize == 0 { pageSize = 20 }
	pages := int32((total + int64(pageSize) - 1) / int64(pageSize))

	return &pb.ListProductsResponse{Products: proto, Total: total, Page: req.Page, Pages: pages}, nil
}

func (h *ProductHandler) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id")
	}
	newStock, err := h.svc.UpdateStock(ctx, id, req.Quantity, product.StockOperation(req.Op))
	if err != nil {
		return nil, toProductGRPCError(err)
	}
	return &pb.UpdateStockResponse{Id: req.Id, Stock: newStock}, nil
}

// ── Mappers ───────────────────────────────────────────────────

func toProtoProduct(p *product.Product) *pb.Product {
	pp := &pb.Product{
		Id:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		Category:    p.Category,
		IsActive:    p.IsActive,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
	if p.Metadata != nil {
		pp.Tags     = p.Metadata.Tags
		pp.Metadata = p.Metadata.Attributes
	}
	return pp
}

func toProductGRPCError(err error) error {
	switch {
	case pkgerrors.IsNotFound(err):
		return status.Errorf(codes.NotFound, err.Error())
	case pkgerrors.IsValidation(err):
		return status.Errorf(codes.InvalidArgument, err.Error())
	case pkgerrors.IsUnauthenticated(err):
		return status.Errorf(codes.Unauthenticated, err.Error())
	case err == product.ErrNotFound:
		return status.Errorf(codes.NotFound, "product not found")
	case err == product.ErrOutOfStock:
		return status.Errorf(codes.FailedPrecondition, "product out of stock")
	default:
		return status.Errorf(codes.Internal, "internal server error")
	}
}
