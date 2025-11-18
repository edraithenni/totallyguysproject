package handlers

import (
    "net/http"
    "strconv"
    "totallyguysproject/internal/models"
	"totallyguysproject/internal/ws"
	"totallyguysproject/internal/banned"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

// DELETE /api/admin/reviews/:id
func AdminDeleteReview(c *gin.Context, db *gorm.DB, hub *ws.Hub) {
    role, _ := c.Get("role")
    if role != "admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
        return
    }

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

    // delete comment votes tied to review
    if err := db.Exec(`
        DELETE FROM comment_votes 
        WHERE comment_id IN (SELECT id FROM comments WHERE review_id = ?)
    `, reviewID).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete votes"})
        return
    }

    // delete comments
    if err := db.Where("review_id = ?", reviewID).
        Unscoped().Delete(&models.Comment{}).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comments"})
        return
    }

    // soft delete review
    if err := db.Unscoped().Delete(&review).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete review"})
        return
    }

	// send websocket notification to author
	hub.Send(review.UserID, map[string]interface{}{
    	"type":      "review_deleted",
    	"text":      "Your review was deleted by a moderator",
    	"review_id": review.ID,
	})

    c.JSON(http.StatusOK, gin.H{"message": "review deleted by admin"})
}

// DELETE /api/admin/comments/:id
func AdminDeleteComment(c *gin.Context, db *gorm.DB, hub *ws.Hub){
    role, _ := c.Get("role")
    if role != "admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
        return
    }

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

    // check if has replies
    var child models.Comment
    if err := db.Unscoped().Where("parent_id = ?", comment.ID).First(&child).Error; err == nil {
        // cannot delete, replace with admin mark
        tx := db.Begin()

        tx.Exec("SET CONSTRAINTS ALL DEFERRED")

        if err := tx.Exec("UPDATE comments SET deleted_at = NOW() WHERE id = ?", comment.ID).Error; err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
            return
        }

        if err := tx.Exec("UPDATE comments SET content = '[deleted by moderator]' WHERE id = ?", comment.ID).Error; err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update content"})
            return
        }
		hub.Send(comment.UserID, map[string]interface{}{
    		"type":       "comment_deleted",
    		"text":       "Your comment was deleted by a moderator",
    		"comment_id": comment.ID,
		})

        tx.Commit()
        c.JSON(http.StatusOK, gin.H{"message": "comment marked deleted by admin"})
        return
    }

    // soft-delete normally
    if err := db.Unscoped().Delete(&comment).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
        return
    }
	// notify author
	hub.Send(comment.UserID, map[string]interface{}{
    	"type":       "comment_deleted",
    	"text":       "Your comment was deleted by a moderator",
    	"comment_id": comment.ID,
	})

    // apply ancestor cleanup
    CleanUpDeletedAncestors(c, db, comment.ParentID)

    c.JSON(http.StatusOK, gin.H{"message": "comment deleted by admin"})
}

func AdminBanUser(c *gin.Context, hub *ws.Hub) {
    uidStr := c.Param("id")
    uid64, _ := strconv.ParseUint(uidStr, 10, 64)
    userID := uint(uid64)

    banned.BanUser(userID)

    hub.Send(userID, map[string]interface{}{
        "type": "banned",
        "text": "Your account has been banned by a moderator",
    })

    c.JSON(http.StatusOK, gin.H{"message": "user banned"})
}

func AdminUnbanUser(c *gin.Context, hub *ws.Hub) {
    uidStr := c.Param("id")
    uid64, _ := strconv.ParseUint(uidStr, 10, 64)
    userID := uint(uid64)

    banned.UnbanUser(userID)

	hub.Send(userID, map[string]interface{}{
        "type": "banned",
        "text": "Your account has been unbanned by a moderator",
    })
    c.JSON(http.StatusOK, gin.H{"message": "user unbanned"})
}

