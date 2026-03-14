package grpc_client

import (
	"context"
	"fmt"

	"github.com/gibbon/finace-dashboard/internal/grpc_client/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


type MLClient struct {
	conn   *grpc.ClientConn
	client pb.CategorizationServiceClient
}


type MLClientConfig struct {
	Host string
	Port string
}

func NewMLClient(cfg MLClientConfig) (*MLClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ML service: %w", err)
	}

	client := pb.NewCategorizationServiceClient(conn)

	return &MLClient{
		conn:   conn,
		client: client,
	}, nil
}


func (c *MLClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}


type CategorizeResult struct {
	CategoryID   int32
	CategoryName string
	Confidence   float32
}

func (c *MLClient) Categorize(ctx context.Context, description string, amount float64, currency string) (*CategorizeResult, error) {
	req := &pb.CategorizeRequest{
		Description: description,
		Amount:      float32(amount),
		Currency:    currency,
	}

	resp, err := c.client.Categorize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("categorization failed: %w", err)
	}

	return &CategorizeResult{
		CategoryID:   resp.CategoryId,
		CategoryName: resp.CategoryName,
		Confidence:   resp.Confidence,
	}, nil
}
