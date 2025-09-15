package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/argon2"
)

type contextKey string

const (
	format = "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	hmacSecret = "V2pkd1pVaDJkV0pHZEZCMVVXUnliZw=="
	defaulExpireTime = 604800 // 1 week
	passwordTime = 1
	passwordMemory = 64 * 1024
	passwordThreads = 4
	passwordKeyLen = 32
	
	UserContextKey = contextKey("user")
)

type Claims struct {
	Name string `json:"name"`
	jwt.StandardClaims
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Name string `json:"name"`
}

func GeneratePassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, passwordTime, passwordMemory, passwordThreads, passwordKeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf(format, argon2.Version, passwordMemory, passwordTime, passwordThreads, b64Salt, b64Hash), nil
}

func ValidatePassword(password, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	keyLen := uint32(len(decodedHash))
	comparisonHash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	return (subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1), nil
}

func CreateJWTToken(user *User) (*TokenResponse, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name":      user.Name,
		"exp": time.Now().Unix() + defaulExpireTime,
	})
	accessToken, err := token.SignedString([]byte(hmacSecret))
	if err != nil {
		return nil, err
	}
	return &TokenResponse{accessToken, user.Name}, nil
}

func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(hmacSecret), nil
	})
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}

func AuthMiddleware(f http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, tok := r.URL.Query()["token"]
		if tok && len(token) == 1 {
			user, err := ValidateToken(token[0])
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
			} else {
				ctx := context.WithValue(r.Context(), UserContextKey, user.Name)
				f(w, r.WithContext(ctx))
			}
		}  else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Please login or provide name"))
		}
	})
}