package jwtAuthentication

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/golang-jwt/jwt/v5"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	"golang.org/x/crypto/bcrypt"
)

//NOTE: bcrypt.DefaultCost  // use default cost of bcrypt otherwise it will slow down

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

const (
	AccessTokenValidity  time.Duration = 14 * 24 * time.Hour // for development keep it long
	RefreshTokenValidity time.Duration = 28 * 24 * time.Hour // for development keep it long
)

// JWT Claims with Email
// Claims is the struct that contains the email of the user
// Email is the email of the user
// RegisteredClaims is the struct that contains the registered claims of the token
// ExpiresAt is the expiration time of the token
type JwtClaims struct {
	Email     string `json:"email"`
	UserId    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// Helper - Hash password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Helper - Check password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Generate JWT tokens
func GenerateToken(authConfigs *configs.AuthConfig, email string, userId snowflake.ID, duration time.Duration, tokenType TokenType) (string, error) {
	claims := &JwtClaims{
		Email:     email,
		UserId:    userId.String(),
		TokenType: string(tokenType),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userId.String(),
			Audience:  jwt.ClaimStrings{authConfigs.JWTAudience},
			Issuer:    authConfigs.JWTIssuer,
		},
	}
	// Create a new token with the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// signed the token with the secret key and return it
	return token.SignedString([]byte(authConfigs.JWTSecret))
}
