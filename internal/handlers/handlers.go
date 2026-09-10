package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/whu-ham/ham-gateway/internal/ham"
)

// HamHandler handles requests to HAM API
type HamHandler struct {
	hamClient ham.ClientInterface
}

// NewHamHandler creates a new HAM handler
func NewHamHandler(client ham.ClientInterface) *HamHandler {
	return &HamHandler{
		hamClient: client,
	}
}

// SearchCourse handles GET /api/v1/external/ham/course/search
// Query params: keyword (required), keyword_type (optional, default 0)
func (h *HamHandler) SearchCourse(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": "keyword is required",
		})
		return
	}

	keywordTypeStr := c.DefaultQuery("keyword_type", "0")
	keywordType, err := strconv.ParseInt(keywordTypeStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": "invalid keyword_type",
		})
		return
	}

	resp, err := h.hamClient.SearchCourse(c.Request.Context(), keyword, int32(keywordType))
	if err != nil {
		log.Printf("[SearchCourse] HAM API call failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "500",
			"message": "Failed to connect to HAM API",
		})
		return
	}

	// Transform response to match gateway format
	courseItems := make([]gin.H, 0, len(resp.Item))
	for _, item := range resp.Item {
		courseItems = append(courseItems, gin.H{
			"name":       item.Value,
			"instructor": "",
			"type":       strconv.Itoa(int(item.Type)),
			"score":      item.Hit,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "00000",
		"message": "Success",
		"data":    courseItems,
	})
}

// GetCourseStat handles GET /api/v1/external/ham/score/stat
// Query params: course_name (required), instructor (required)
func (h *HamHandler) GetCourseStat(c *gin.Context) {
	courseName := c.Query("course_name")
	instructor := c.Query("instructor")

	if courseName == "" || instructor == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": "course_name and instructor are required",
		})
		return
	}

	resp, err := h.hamClient.GetCourseScoreItem(c.Request.Context(), courseName, instructor)
	if err != nil {
		log.Printf("[GetCourseStat] HAM API call failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "500",
			"message": "Failed to connect to HAM API",
		})
		return
	}

	// HAM returns a nil Item when the (course_name, instructor) pair has no
	// score data — e.g. unknown course-name variants such as "X（创）". Treat
	// this as an empty/not-found result instead of dereferencing nil, which
	// previously panicked and surfaced as a 500.
	if resp == nil || resp.Item == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    "00000",
			"message": "Success",
			"data": gin.H{
				"name":       "",
				"instructor": "",
				"average":    0,
				"total":      0,
				"range":      []gin.H{},
			},
		})
		return
	}

	// Transform response to match gateway format
	scoreRanges := make([]gin.H, 0, len(resp.Item.Range))
	for _, r := range resp.Item.Range {
		scoreRanges = append(scoreRanges, gin.H{
			"from":  r.From,
			"to":    r.To,
			"total": r.Total,
			"color": r.Color,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "00000",
		"message": "Success",
		"data": gin.H{
			"name":       resp.Item.Name,
			"instructor": resp.Item.Instructor,
			"average":    resp.Item.Average,
			"total":      resp.Item.Total,
			"range":      scoreRanges,
		},
	})
}

// GetCourseStatsByName handles GET /api/v1/external/ham/score/by-course.
// Query params: course_name (required), page_num (optional), page_size (optional).
func (h *HamHandler) GetCourseStatsByName(c *gin.Context) {
	courseName := c.Query("course_name")
	if courseName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "course_name is required"})
		return
	}

	pageNum, err := parseNonNegativeInt32(c.DefaultQuery("page_num", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "invalid page_num"})
		return
	}
	pageSize, err := parseNonNegativeInt32(c.DefaultQuery("page_size", "30"))
	if err != nil || pageSize == 0 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "page_size must be between 1 and 100"})
		return
	}

	resp, err := h.hamClient.GetCourseScoresByCourseName(c.Request.Context(), courseName, pageNum, pageSize)
	if err != nil {
		log.Printf("[GetCourseStatsByName] HAM API call failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": "Failed to connect to HAM API"})
		return
	}
	if resp == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": "00000", "message": "Success",
			"data": gin.H{"items": []gin.H{}, "page_num": pageNum, "has_more": false},
		})
		return
	}

	items := make([]gin.H, 0, len(resp.Item))
	for _, item := range resp.Item {
		if item == nil {
			continue
		}
		ranges := make([]gin.H, 0, len(item.Range))
		for _, scoreRange := range item.Range {
			ranges = append(ranges, gin.H{"from": scoreRange.From, "to": scoreRange.To, "total": scoreRange.Total, "color": scoreRange.Color})
		}
		items = append(items, gin.H{
			"id": item.Id, "name": item.Name, "instructor": item.Instructor,
			"average": item.Average, "total": item.Total, "range": ranges,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": "00000", "message": "Success",
		"data": gin.H{"items": items, "page_num": resp.PageNum, "has_more": resp.HasMore},
	})
}

func parseNonNegativeInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("value must be non-negative")
	}
	return int32(parsed), nil
}
