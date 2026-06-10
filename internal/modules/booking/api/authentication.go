package bookingApi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	jwtAuthentication "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/lib/jwtAuth"
)

func (apiHandler *ApiHandler) RegisterAuthRoutes(rg *gin.RouterGroup) {
	// Public
	authPublicRg := rg.Group("/public/auth")
	authPublicRg.POST("signup", apiHandler.signUp)
	authPublicRg.POST("signin", apiHandler.signIn)

	// Protected
	authRg := rg.Group("/auth", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))
	authRg.POST("refresh", apiHandler.getAccessTokenFromRefresh)
	authRg.POST("signout", apiHandler.signout)
}

// Set tokens in httpOnly cookies
func _setRefreshTokenCookies(c *gin.Context, accessToken string, accessTokenValidity int, refreshToken string, refreshTokenValidity int) {
	// Access Token Cookie (short-lived)
	c.SetSameSite(http.SameSiteStrictMode) // CSRF protection
	c.SetCookie(
		"accessToken",
		accessToken,
		accessTokenValidity,
		"",
		"",
		false, // LOOKUP: secure (true in production)
		false, // LOOKUP: httpOnly
	)

	// Refresh Token Cookie (long-lived)
	if refreshTokenValidity > 0 {
		c.SetCookie(
			"refreshToken",
			refreshToken,
			refreshTokenValidity,
			"/api/auth/refresh",
			"",
			false, // LOOKUP: secure (true in production)
			false, // LOOKUP: httpOnly
		)
	}
}

// Clear cookies on logout
func _clearTokenCookies(c *gin.Context) {
	c.SetCookie("accessToken", "", -1, "/", "", true, true)
	c.SetCookie("refreshToken", "", -1, "/api/auth/refresh", "", true, true)
}

// Generate the user session with access and refresh tokens set in headers
func _generateUserTokens(ctx *gin.Context, authConfigs *configs.AuthConfigurations, user *models.User) (access string, refresh string, err error) {
	// Generate Access Token
	refreshToken, err := jwtAuthentication.GenerateToken(authConfigs, user.Email, user.ID, jwtAuthentication.RefreshTokenValidity, jwtAuthentication.RefreshToken)
	if err != nil {
		return "", "", err
	}

	accessToken, err := jwtAuthentication.GenerateToken(authConfigs, user.Email, user.ID, jwtAuthentication.AccessTokenValidity, jwtAuthentication.AccessToken)
	if err != nil {
		return "", "", err
	}

	// Set new cookies
	_setRefreshTokenCookies(ctx, accessToken, int(jwtAuthentication.AccessTokenValidity), refreshToken, int(jwtAuthentication.RefreshTokenValidity.Seconds()))
	return accessToken, refreshToken, nil
}

// SignUpRequest handles exactly what the user sends
type SignRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=3,max=72"`
	ImageUrl string `json:"image_url" binding:"omitempty,url,max=255"`
}

// User sign-up
func (apiHandler *ApiHandler) signUp(ctx *gin.Context) {
	var incomingUser SignRequest

	// Bind the json with the user struct
	if err := ctx.ShouldBindJSON(&incomingUser); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Hash Password
	hashedPassword, err := jwtAuthentication.HashPassword(incomingUser.Password)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	var createdUser = &models.User{
		Email:    incomingUser.Email,
		Password: hashedPassword,
		Name:     incomingUser.Name,
		ImageUrl: incomingUser.ImageUrl,
	}

	// Create account
	appErr := apiHandler.cacheStore.CreateUser(ctx, createdUser)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	// Generate the user session with tokens
	accessToken, refreshToken, err := _generateUserTokens(ctx, apiHandler.authConfigs, createdUser)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Send success response with token
	utils.RespondWithSuccess(ctx, http.StatusCreated, gin.H{
		"message":      "Signup Successful",
		"user":         createdUser,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

// User Sign-in
func (apiHandler *ApiHandler) signIn(ctx *gin.Context) {
	var incomingUser SignRequest
	// Bind incoming JSON to input struct
	if err := ctx.ShouldBindJSON(&incomingUser); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Find user by email
	user, appErr := apiHandler.cacheStore.GetUserByEmail(ctx, incomingUser.Email)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	// Check password by comparing input password to stored hashed password
	if !jwtAuthentication.CheckPasswordHash(incomingUser.Password, user.Password) {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid password")
		return
	}

	accessToken, refreshToken, err := _generateUserTokens(ctx, apiHandler.authConfigs, user)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Send success response with token
	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":      "Sign-in Successful",
		"user_id":      user.ID,
		"user_email":   user.Email,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

// User Sign-out
func (apiHandler *ApiHandler) signout(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refreshToken")
	if err != nil {
		// Even if no cookie, clear any existing tokens
		_clearTokenCookies(c)
		utils.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
		return
	}

	// Parse token
	token, err := jwt.ParseWithClaims(refreshToken, &jwtAuthentication.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(apiHandler.authConfigs.JWTSecret), nil
	})

	if err != nil {
		_clearTokenCookies(c)
		utils.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
		return
	}

	_, claimsOk := token.Claims.(*jwtAuthentication.JwtClaims)
	if claimsOk == false {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid token")
		return
	}

	// Clear cookies
	_clearTokenCookies(c)

	utils.RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": " Logout successful",
	})
}

// Get the access token using the refresh Token
func (apiHandler *ApiHandler) getAccessTokenFromRefresh(ctx *gin.Context) {

	email := ctx.GetString(constants.UserEmailKey)

	// Check if user exists in database and matches
	user, appErr := apiHandler.cacheStore.GetUserByEmail(ctx, email)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	// Generate NEW access token (and optionally new refresh token - token rotation)
	newAccessToken, err := jwtAuthentication.GenerateToken(apiHandler.authConfigs, user.Email, user.ID, jwtAuthentication.AccessTokenValidity, jwtAuthentication.AccessToken)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":     "Access Token refreshed successfully",
		"accessToken": newAccessToken,
	})
}
