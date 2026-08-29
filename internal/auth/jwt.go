package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid" // 需要安裝這個套件來產生唯一 ID: go get github.com/google/uuid
)

// 實務上應從環境變數 (os.Getenv) 讀取，此處為範例硬編碼
var secretKey = []byte("flashchat_secret_key_2026")

// 實務上可以為 Refresh Token 準備另一把獨立的 Secret Key，這裡為求簡潔先共用
var refreshSecretKey = []byte("flashchat_refresh_secret_2026")

// Claims 定義 Token 內含的資料結構
type Claims struct {
	UID                  string `json:"uid"` // 👈 新增：使用者的唯一標識符 (如 DB 中的 ID)`
	Name                 string `json:"name"`
	TokenType            string `json:"token_type"` // 👈 新增：用來區分 "access" 或 "refresh"
	jwt.RegisteredClaims        // 嵌入官方的標準欄位
}

// GenerateTokens 一次核發「短效通行證」與「長效更新證」
// feature: 密鑰輪換功能 參數新增 ctx 和 km (倉管員)、 uid
func GenerateToken(ctx context.Context, km *KeyManager, uid string, name string) (string, string, error) {
	// 1. 跟倉管員拿目前最新服役的主鑰匙
	kid, secretStr, err := km.GetPrimarySecret(ctx)
	if err != nil {
		return "", "", fmt.Errorf("無法取得主密鑰: %v", err)
	}

	// 將字串鑰匙轉為位元組，給 JWT 套件使用
	secretKey := []byte(secretStr)

	// 產生一個唯一的 JTI
	//tokenID := uuid.New().String()

	// 2. 產生 Access Token (壽命：15 分鐘)
	accessTokenID := uuid.New().String()
	accessClaims := &Claims{
		UID:       uid, // 👈 寫入 UID
		Name:      name,
		TokenType: "access", // 🌟 貼上通行證標籤
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        accessTokenID, // 🌟 這裡就是塞入 jti 的地方
		},
	}

	// 步驟 A：建立 Token 結構體 (此時還是個物件，不是字串)
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	// 步驟 B：🌟 在轉成字串前，先設定 Header 加上 kid
	accessTokenObj.Header["kid"] = kid // 🌟 放入動態取得的 kid

	accessToken, err := accessTokenObj.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	// ==========================================
	// 3. 處理 Refresh Token (壽命：7 天)
	// ==========================================

	// 💡 建議：Refresh Token 也給它一個獨立的 ID，未來若需要單獨拉黑它會很好用
	refreshTokenID := uuid.New().String()
	refreshClaims := &Claims{
		UID:       uid, // 👈 修正：必須補上 UID！否則 Refresh 換發時解出的 claims.UID 會是空字串
		Name:      name,
		TokenType: "refresh", // 🌟 貼上換發憑證標籤
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Token 到期時間
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        refreshTokenID, //🌟 Refresh Token 的 jti                      //Token 出產時間
		},
	}

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenObj.Header["kid"] = kid // 🌟 Refresh Token 同樣動態取得 kid

	refreshToken, err := refreshTokenObj.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ValidateToken 驗證 Access Token
func ValidateToken(ctx context.Context, km *KeyManager, tokenString string) (*Claims, error) {
	// 呼叫共用的底層邏輯
	claims, err := parseToken(ctx, km, tokenString)
	if err != nil {
		return nil, err
	}
	// 🌟 拿到解碼結果後，多加一道防線：檢查標籤是不是 access
	if claims.TokenType != "access" {
		return nil, fmt.Errorf("無效的 Token 類型：期望 access，卻得到 %s", claims.TokenType)
	}
	return claims, nil
}

// ValidateRefreshToken 驗證 Refresh Token
func ValidateRefreshToken(ctx context.Context, km *KeyManager, tokenString string) (*Claims, error) {
	//  呼叫共用的底層邏輯
	claims, err := parseToken(ctx, km, tokenString)
	if err != nil {
		return nil, err
	}
	// 🌟 拿到解碼結果後，多加一道防線：檢查標籤是不是 refresh
	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("嚴重安全性風險：非 refresh token 嘗試進行換發動作")
	}
	return claims, nil
}

// 內部共用的解析邏輯
func parseToken(ctx context.Context, km *KeyManager, tokenString string) (*Claims, error) {
	// 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 1. 安全性檢查：確保駭客沒有竄改簽章演算法 (例如把 HS256 改成無簽章的 "none")
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("未知的簽章方法: %v", token.Header["alg"])
		}
		// 2. 從 Token 的 Header 中拿出 kid
		kidInterface, ok := token.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("Token 缺少 kid 標籤")
		}

		// 確保 kid 是字串型別
		kid, ok := kidInterface.(string)
		if !ok {
			return nil, fmt.Errorf("kid 格式錯誤")
		}

		//3. 🌟 拿著這個 kid，去請倉管員從 Redis 翻出對應的鑰匙！
		secretStr, err := km.GetSecretByKeyID(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("無效的密鑰 ID: %v", err)
		}

		// 4. 回傳轉換成位元組的密鑰，讓套件進行數學驗證
		return []byte(secretStr), nil

	})

	if err != nil {
		return nil, err
	}

	//token.valid 這是 jwt 套件給我們的驗證報告。套件會去檢查：「簽章是否正確？」、「這個 Token 過期 (exp) 了沒有？」。如果一切合法未過期，這項才會是 true。
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("無效的 Token")
}
