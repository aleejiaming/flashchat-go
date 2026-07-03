package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 實務上應從環境變數 (os.Getenv) 讀取，此處為範例硬編碼
var secretKey = []byte("flashchat_secret_key_2026")

// Claims 定義 Token 內含的資料結構
type Claims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

func GenerateToken(name string) (string, error) {
	claims := &Claims{
		Name: name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24小時後過期
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidateToken 驗證 JWT 並回傳使用者名稱
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.Name, nil
	}
	return "", errors.New("無效的 Token")
}
