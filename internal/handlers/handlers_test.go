package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	pb "github.com/whu-ham/ham-gateway/gen/proto"
)

// MockHamClient is a mock implementation of ham.ClientInterface
type MockHamClient struct {
	mock.Mock
}

func (m *MockHamClient) SearchCourse(ctx context.Context, keyword string, keywordType int32) (*pb.SearchCourseResponse, error) {
	args := m.Called(ctx, keyword, keywordType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.SearchCourseResponse), args.Error(1)
}

func (m *MockHamClient) GetCourseScoreItem(ctx context.Context, courseName, instructor string) (*pb.GetCourseScoreItemResponse, error) {
	args := m.Called(ctx, courseName, instructor)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetCourseScoreItemResponse), args.Error(1)
}

func (m *MockHamClient) GetCourseScoresByCourseName(ctx context.Context, courseName string, pageNum, pageSize int32) (*pb.GetCourseScoresByCourseNameResponse, error) {
	args := m.Called(ctx, courseName, pageNum, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetCourseScoresByCourseNameResponse), args.Error(1)
}

func (m *MockHamClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestSearchCourse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		keyword        string
		keywordType    string
		mockResponse   *pb.SearchCourseResponse
		mockError      error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:        "Success",
			keyword:     "数学",
			keywordType: "1",
			mockResponse: &pb.SearchCourseResponse{
				Item: []*pb.SearchCourseHitItem{
					{Type: pb.CourseScoreItemType_COURSE_SCORE_ITEM_TYPE_COURSE_NAME, Value: "高等数学", Hit: "高等<em>数学</em>"},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCode:   "00000",
		},
		{
			name:           "Missing keyword",
			keyword:        "",
			keywordType:    "0",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400",
		},
		{
			name:           "Invalid keyword_type",
			keyword:        "test",
			keywordType:    "invalid",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400",
		},
		{
			name:           "HAM API error",
			keyword:        "test",
			keywordType:    "0",
			mockResponse:   nil,
			mockError:      errors.New("connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockHamClient)
			if tt.keyword != "" && tt.keywordType != "invalid" {
				mockClient.On("SearchCourse", mock.Anything, tt.keyword, mock.AnythingOfType("int32")).
					Return(tt.mockResponse, tt.mockError)
			}

			handler := &HamHandler{hamClient: mockClient}

			r := gin.New()
			r.GET("/search", handler.SearchCourse)

			url := "/search"
			if tt.keyword != "" || tt.keywordType != "" {
				url += "?keyword=" + tt.keyword + "&keyword_type=" + tt.keywordType
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedCode)

			if tt.keyword != "" && tt.keywordType != "invalid" {
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestGetCourseStat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		courseName     string
		instructor     string
		mockResponse   *pb.GetCourseScoreItemResponse
		mockError      error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:       "Success",
			courseName: "高等数学",
			instructor: "张三",
			mockResponse: &pb.GetCourseScoreItemResponse{
				Item: &pb.CourseScoreItem{
					Name:       "高等数学",
					Instructor: "张三",
					Average:    85.5,
					Total:      100,
					Range: []*pb.CourseScoreRangeResponseItem{
						{From: 90, To: 100, Total: 30, Color: "#4CAF50"},
						{From: 80, To: 89, Total: 40, Color: "#8BC34A"},
					},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCode:   "00000",
		},
		{
			name:           "Missing course_name",
			courseName:     "",
			instructor:     "张三",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400",
		},
		{
			name:           "Missing instructor",
			courseName:     "高等数学",
			instructor:     "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400",
		},
		{
			name:           "HAM API error",
			courseName:     "test",
			instructor:     "test",
			mockResponse:   nil,
			mockError:      errors.New("connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "500",
		},
		{
			// HAM returns a response with a nil Item for unknown course-name
			// variants (e.g. "高等数学A1（创）"). The handler must not dereference
			// nil (which panicked -> 500); it returns an empty 200 that callers
			// treat as not-found.
			name:           "Nil item (course not found) returns empty 200",
			courseName:     "高等数学A1（创）",
			instructor:     "刘丁酉",
			mockResponse:   &pb.GetCourseScoreItemResponse{Item: nil},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCode:   "00000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockHamClient)
			if tt.courseName != "" && tt.instructor != "" {
				mockClient.On("GetCourseScoreItem", mock.Anything, tt.courseName, tt.instructor).
					Return(tt.mockResponse, tt.mockError)
			}

			handler := &HamHandler{hamClient: mockClient}

			r := gin.New()
			r.GET("/stat", handler.GetCourseStat)

			url := "/stat?course_name=" + tt.courseName + "&instructor=" + tt.instructor

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedCode)

			if tt.courseName != "" && tt.instructor != "" {
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestGetCourseStatsByName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name         string
		url          string
		response     *pb.GetCourseScoresByCourseNameResponse
		err          error
		status       int
		code         string
		expectCall   bool
		expectedPage int32
		expectedSize int32
	}{
		{
			name: "success", url: "/scores?course_name=Math&page_num=1&page_size=20",
			response: &pb.GetCourseScoresByCourseNameResponse{
				PageNum: 1,
				Item:    []*pb.CourseScoreItem{{Name: "Math", Instructor: "Smith", Average: 85.5, Total: 20}},
			},
			status: http.StatusOK, code: "00000", expectCall: true, expectedPage: 1, expectedSize: 20,
		},
		{
			name: "missing course name", url: "/scores", status: http.StatusBadRequest, code: "400",
		},
		{
			name: "invalid page size", url: "/scores?course_name=Math&page_size=101", status: http.StatusBadRequest, code: "400",
		},
		{
			name: "upstream error", url: "/scores?course_name=Math", err: errors.New("connection failed"),
			status: http.StatusInternalServerError, code: "500", expectCall: true, expectedPage: 0, expectedSize: 30,
		},
		{
			// Defensive: a nil response with no error must not panic; it yields
			// an empty page.
			name: "nil response", url: "/scores?course_name=Math", response: nil, err: nil,
			status: http.StatusOK, code: "00000", expectCall: true, expectedPage: 0, expectedSize: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockHamClient)
			if tt.expectCall {
				mockClient.On("GetCourseScoresByCourseName", mock.Anything, mock.Anything, tt.expectedPage, tt.expectedSize).
					Return(tt.response, tt.err)
			}
			r := gin.New()
			r.GET("/scores", (&HamHandler{hamClient: mockClient}).GetCourseStatsByName)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
			assert.Contains(t, w.Body.String(), tt.code)
			mockClient.AssertExpectations(t)
		})
	}
}
