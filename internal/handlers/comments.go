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
	if err := db.Unscoped().Where("review_id = ?", reviewID).
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
	if err := db.Unscoped().Where("parent_id = ?", comment.ID).First(&child).Error; err == nil {
		tx := db.Begin()

		tx.Exec("SET CONSTRAINTS ALL DEFERRED")

		if err := tx.Exec("UPDATE comments SET deleted_at = NOW() WHERE id = ?", comment.ID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
			return
		}

		if err := tx.Exec("UPDATE comments SET content = '[deleted]' WHERE id = ?", comment.ID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark content"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "comment marked deleted"})
		return
	}

	// otherwise soft-delete
	if err := db.Unscoped().Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}
	CleanUpDeletedAncestors(c, db, comment.ParentID)
	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

func CleanUpDeletedAncestors(c *gin.Context, db *gorm.DB, parentID *uint) {
	if parentID == nil {
		return
	}

	var parent models.Comment
	if err := db.Unscoped().First(&parent, *parentID).Error; err != nil {
		return
	}

	if parent.Content != "[deleted]" {
		return
	}

	var count int64
	if err := db.Unscoped().Model(&models.Comment{}).
		Where("parent_id = ?", parent.ID).
		Count(&count).Error; err != nil {
		return
	}

	if count == 0 {
		if err := db.Unscoped().Delete(&parent).Error; err != nil {
			return
		}
		CleanUpDeletedAncestors(c, db, parent.ParentID)
	}
}

// POST /api/comments/:id/vote
func VoteComment(c *gin.Context, db *gorm.DB) {
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
	commentID := uint(cid64)

	var req struct {
		Action string `json:"action"` // "up", "down", "remove"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var comment models.Comment
	if err := db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	var existing models.CommentVote
	err = db.Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error

	switch req.Action {
	case "up":
		if err == gorm.ErrRecordNotFound {
			db.Create(&models.CommentVote{UserID: userID, CommentID: commentID, Value: 1})
			comment.Value++
		} else if existing.Value == -1 {
			existing.Value = 1
			db.Save(&existing)
			comment.Value += 2
		} else if existing.Value == 1 {
			c.JSON(http.StatusOK, gin.H{"value": comment.Value})
			return
		}

	case "down":
		if err == gorm.ErrRecordNotFound {
			db.Create(&models.CommentVote{UserID: userID, CommentID: commentID, Value: -1})
			comment.Value--
		} else if existing.Value == 1 {
			existing.Value = -1
			db.Save(&existing)
			comment.Value -= 2
		} else if existing.Value == -1 {
			c.JSON(http.StatusOK, gin.H{"value": comment.Value})
			return
		}

	case "remove":
		if err == nil {
			// cancel vote
			comment.Value -= existing.Value
			db.Delete(&existing)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no vote to remove"})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}

	if err := db.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment value"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"value": comment.Value})
}
