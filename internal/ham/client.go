package ham

import (
	"context"
	"crypto/tls"
	"fmt"

	pb "github.com/whu-ham/ham-gateway/gen/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	conn         *grpc.ClientConn
	scoreClient  pb.GetCourseScoreServiceClient
	searchClient pb.SearchCourseServiceClient
	appID        string
	appSecret    string
}

func NewClient(baseURL, appID, appSecret, certPath, keyPath string) (*Client, error) {
	// Load client certificate and key for mTLS
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load x509 key pair: %w", err)
	}

	// Create TLS credentials
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	creds := credentials.NewTLS(tlsConfig)

	// Create gRPC connection with mTLS
	conn, err := grpc.NewClient(
		baseURL,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	return &Client{
		conn:         conn,
		scoreClient:  pb.NewGetCourseScoreServiceClient(conn),
		searchClient: pb.NewSearchCourseServiceClient(conn),
		appID:        appID,
		appSecret:    appSecret,
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// addAuth adds authentication headers to the context
func (c *Client) addAuth(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{
		"x-app-id":       c.appID,
		"x-app-secret":   c.appSecret,
		"x-request-type": "open-app", // Required by ham-backend-go/internal/pkg/contextx/context.go
	})
	return metadata.NewOutgoingContext(ctx, md)
}

// SearchCourse calls HAM API to search for courses
func (c *Client) SearchCourse(ctx context.Context, keyword string, keywordType int32) (*pb.SearchCourseResponse, error) {
	ctx = c.addAuth(ctx)

	req := &pb.SearchCourseRequest{
		Keyword:     keyword,
		KeywordType: pb.SearchCourseKeywordType(keywordType),
	}

	resp, err := c.searchClient.SearchCourse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search course failed: %w", err)
	}

	return resp, nil
}

// GetCourseScoreItem calls HAM API to get course score statistics
func (c *Client) GetCourseScoreItem(ctx context.Context, courseName, instructor string) (*pb.GetCourseScoreItemResponse, error) {
	ctx = c.addAuth(ctx)

	req := &pb.GetCourseScoreItemRequest{
		CourseName: &courseName,
		Instructor: &instructor,
	}

	resp, err := c.scoreClient.GetCourseScoreItem(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get course score item failed: %w", err)
	}

	return resp, nil
}
