package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 實務上應從環境變數 (os.Getenv) 讀取，此處為範例硬編碼
var secretKey = []byte("flashchat_secret_key_2026")

// 實務上可以為 Refresh Token 準備另一把獨立的 Secret Key，這裡為求簡潔先共用
var refreshSecretKey = []byte("flashchat_refresh_secret_2026")

// Claims 定義 Token 內含的資料結構
type Claims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

// GenerateTokens 一次核發「短效通行證」與「長效更新證」
func GenerateToken(name string) (string, string, error) {
	// 1. 產生 Access Token (壽命：15 分鐘)
	accessClaims := &Claims{
		Name: name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	// 2. 產生 Refresh Token (壽命：7 天)
	refreshClaims := &Claims{
		Name: name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Token 到期時間
			IssuedAt:  jwt.NewNumericDate(time.Now()),                         //Token 出產時間
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(refreshSecretKey)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ValidateToken 驗證 Access Token
func ValidateToken(tokenString string) (string, error) {
	return parseToken(tokenString, secretKey)
}

// ValidateRefreshToken 驗證 Refresh Token
func ValidateRefreshToken(tokenString string) (string, error) {
	return parseToken(tokenString, refreshSecretKey)
}

// 內部共用的解析邏輯
func parseToken(tokenString string, secret []byte) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return "", err
	} //token.valid 這是 jwt 套件給我們的驗證報告。套件會去檢查：「簽章是否正確？」、「這個 Token 過期 (exp) 了沒有？」。如果一切合法未過期，這項才會是 true。
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.Name, nil
	}
	return "", errors.New("無效的 Token")
}
