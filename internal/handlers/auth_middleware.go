package handlers

import (
    "net/http"
    "strings"
    "totallyguysproject/internal/utils"

    "github.com/gin-gonic/gin"
)

func AuthMiddleware(optional bool) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := ""
        
        // try get token from cookie
        cookie, err := c.Cookie("token")
        if err == nil {
            token = cookie
        }

        // If cookie is missing, check the Authorization header
        if token == "" {
            authHeader := c.GetHeader("Authorization")
            if authHeader != "" {
                parts := strings.Split(authHeader, " ")
                if len(parts) == 2 && parts[0] == "Bearer" {
                    token = parts[1]
                }
            }
        }
        //If token is still empty, return unauthorized/guest
        if token == "" {
			if optional {
                // guest
                c.Set("userID", uint(0))
                c.Set("role", "guest")
                c.Next()
                return
            }
            c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
            c.Abort()
            return
        }
        //Verify the token
        claims, err := utils.ParseJWT(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // to context
		uidFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}
		c.Set("userID", uint(uidFloat))

		if email, ok := claims["email"].(string); ok {
			c.Set("email", email)
		}
        // Continue request processing
        c.Next()
    }
}
