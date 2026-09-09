package ham

import (
	"context"

	pb "github.com/whu-ham/ham-gateway/gen/proto"
)

// ClientInterface defines the interface for HAM API client
// This allows for easy mocking in tests
type ClientInterface interface {
	SearchCourse(ctx context.Context, keyword string, keywordType int32) (*pb.SearchCourseResponse, error)
	GetCourseScoreItem(ctx context.Context, courseName, instructor string) (*pb.GetCourseScoreItemResponse, error)
	GetCourseScoresByCourseName(ctx context.Context, courseName string, pageNum, pageSize int32) (*pb.GetCourseScoresByCourseNameResponse, error)
	Close() error
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)
