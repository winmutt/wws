package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"wws/api/internal/handlers"
)

const (
	RoleOwner  = handlers.RoleOwner
	RoleAdmin  = handlers.RoleAdmin
	RoleMember = handlers.RoleMember
	RoleViewer = handlers.RoleViewer
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
	OrgIDKey    contextKey = "org_id"
)

type UserInfo struct {
	UserID int
	Role   string
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Missing session token", http.StatusUnauthorized)
			return
		}

		if cookie.Value == "" {
			http.Error(w, "Empty session token", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sessionInfo, err := handlers.ValidateSession(ctx, cookie.Value)
		if err != nil {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, UserIDKey, sessionInfo.UserID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userIDVal := r.Context().Value(UserIDKey)
			if userIDVal == nil {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			userID, ok := userIDVal.(int)
			if !ok {
				http.Error(w, "Invalid user ID", http.StatusInternalServerError)
				return
			}

			orgIDVal := r.Context().Value(OrgIDKey)
			if orgIDVal == nil {
				http.Error(w, "Organization ID not provided", http.StatusBadRequest)
				return
			}

			orgID, ok := orgIDVal.(int)
			if !ok {
				http.Error(w, "Invalid organization ID", http.StatusInternalServerError)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			member, err := handlers.GetMemberByUserAndOrg(ctx, userID, orgID)
			if err != nil {
				http.Error(w, "Failed to get member", http.StatusInternalServerError)
				return
			}

			if member == nil {
				http.Error(w, "Not a member of this organization", http.StatusForbidden)
				return
			}

			roleAllowed := false
			for _, role := range roles {
				if member.Role == role {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			ctx = context.WithValue(ctx, UserRoleKey, member.Role)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(r *http.Request) (int, bool) {
	userIDVal := r.Context().Value(UserIDKey)
	if userIDVal == nil {
		return 0, false
	}
	userID, ok := userIDVal.(int)
	return userID, ok
}

func GetUserRole(r *http.Request) (string, bool) {
	roleVal := r.Context().Value(UserRoleKey)
	if roleVal == nil {
		return "", false
	}
	role, ok := roleVal.(string)
	return role, ok
}

func GetOrgID(r *http.Request) (int, bool) {
	orgIDVal := r.Context().Value(OrgIDKey)
	if orgIDVal == nil {
		return 0, false
	}
	orgID, ok := orgIDVal.(int)
	return orgID, ok
}

func ExtractOrgIDFromQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgIDStr := r.URL.Query().Get("organization_id")
		if orgIDStr == "" {
			http.Error(w, "Missing organization_id parameter", http.StatusBadRequest)
			return
		}

		var orgID int
		_, err := fmt.Sscanf(orgIDStr, "%d", &orgID)
		if err != nil || orgID == 0 {
			http.Error(w, "Invalid organization_id parameter", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), OrgIDKey, orgID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
