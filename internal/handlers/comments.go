package handlers

import (
	"net/http"
	"strconv"
	"totallyguysproject/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// POST /api/reviews/:id/comments
func CreateComment(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	reviewIDStr := c.Param("id")
	rid64, err := strconv.ParseUint(reviewIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	reviewID := uint(rid64)

	var review models.Review
	if err := db.First(&review, reviewID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}

	var req struct {
		Content  string `json:"content"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty or invalid content"})
		return
	}

	if len(req.Content) > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content too long"})
		return
	}

	comment := models.Comment{
		ReviewID: reviewID,
		UserID:   userID,
		Content:  req.Content,
		ParentID: req.ParentID,
		Value:    0,
	}

	if err := db.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	db.Preload("User").First(&comment, comment.ID)

	c.JSON(http.StatusCreated, comment)
}

// GET /api/reviews/:id/comments
func GetCommentsForReview(c *gin.Context, db *gorm.DB) {
	reviewIDStr := c.Param("id")
	rid64, err := strconv.ParseUint(reviewIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	reviewID := uint(rid64)

	var comments []models.Comment
	if err := db.Where("review_id = ?", reviewID).
		Order("created_at ASC").
		Preload("User").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load comments"})
		return
	}

	tree := buildCommentTree(comments)
	c.JSON(http.StatusOK, tree)
}

// recursively builds a hierarchical comment structure
// by assigning replies to their parent comments based on ParentID.
func buildCommentTree(all []models.Comment) []*models.Comment {
	m := make(map[uint]*models.Comment, len(all))
	for i := range all {
		all[i].Replies = nil
		m[all[i].ID] = &all[i]
	}

	var roots []*models.Comment
	for i := range all {
		c := &all[i]
		if c.ParentID != nil {
			if parent, ok := m[*c.ParentID]; ok {
				parent.Replies = append(parent.Replies, c)
			} else {
				roots = append(roots, c)
			}
		} else {
			roots = append(roots, c)
		}
	}
	return roots
}

// PUT /api/comments/:id
func UpdateComment(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	cidStr := c.Param("id")
	cid64, err := strconv.ParseUint(cidStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}
	cid := uint(cid64)

	var comment models.Comment
	if err := db.First(&comment, cid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	if comment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your comment"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty or invalid content"})
		return
	}

	comment.Content = req.Content
	if err := db.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment"})
		return
	}

	db.Preload("User").First(&comment, comment.ID)
	c.JSON(http.StatusOK, comment)
}

// DELETE /api/comments/:id
// Behavior rule: if a comment has replies, don't delete it —
// replace its content with "[deleted]"; otherwise perform a soft delete.
func DeleteComment(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	cidStr := c.Param("id")
	cid64, err := strconv.ParseUint(cidStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}
	cid := uint(cid64)

	var comment models.Comment
	if err := db.First(&comment, cid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	if comment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your comment"})
		return
	}

	// check for replies
	var child models.Comment
	if err := db.Where("parent_id = ?", comment.ID).First(&child).Error; err == nil {
		// replies exist — replace content with "[deleted]"
		comment.Content = "[deleted]"
		if err := db.Save(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark deleted"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "comment marked deleted"})
		return
	}

	// otherwise soft-delete
	if err := db.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

// POST /api/comments/:id/vote
func VoteComment(c *gin.Context, db *gorm.DB) {}
