package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	// 準備一個空的 Context 給測試使用
	ctx := context.Background()

	// 在記憶體中極速啟動一個假 Redis (測試完自動關閉)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("啟動 miniredis 失敗: %v", err)
	}
	defer mr.Close()

	//  建立連向假 Redis 的 Client
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 順利印出 log 並拿到合法的 testKM 實體！
	testKM := NewKeyManager(rdb)

	// 測試情境 1：正常的 Token 簽發與解析 (Round-trip Test)
	t.Run("正常核發與驗證", func(t *testing.T) {

		testUser := "Mike_Testing"
		testUID := "999" // 🌟 新增：測試用的 UID

		// 1. 產生 Token
		token, _, err := GenerateToken(ctx, testKM, testUID, testUser)
		if err != nil {
			t.Fatalf("產生 Token 失敗: %v", err)
		}

		if token == "" {
			t.Error("預期拿到 token 字串，卻拿到空字串")
		}

		// 2. 驗證剛剛產生的 Token
		parsedClaims, err := ValidateToken(ctx, testKM, token)
		if err != nil {
			t.Fatalf("驗證 Token 失敗: %v", err)
		}

		// 3. 檢查解碼出來的名字跟原本是不是一樣
		if parsedClaims.Name != testUser {
			t.Errorf("解碼名稱錯誤：期望得到 %s，卻得到 %s", testUser, parsedClaims.Name)
		}
	})

	// 測試情境 2：防禦偽造或篡改的 Token
	t.Run("防禦篡改的Token", func(t *testing.T) {
		testUser := "Hacker"
		testUID := "777"

		token, _, _ := GenerateToken(ctx, testKM, testUID, testUser)

		// 駭客嘗試破壞/竄改 token 的內容 (隨便把一段字串轉成大寫)
		tamperedToken := strings.ToUpper(token)

		// 驗證這個被改過的 Token
		_, err := ValidateToken(ctx, testKM, tamperedToken)

		// 我們「期望」這裡必須要有 error (因為這是不合法的 Token)
		if err == nil {
			t.Error("系統被攻破了！篡改過的 Token 竟然驗證成功")
		}
	})

	// 測試情境 3：完全無效的亂碼 Token
	t.Run("無效格式防呆", func(t *testing.T) {
		fakeToken := "this.is.not.a.real.jwt.token"

		_, err := ValidateToken(ctx, testKM, fakeToken)

		if err == nil {
			t.Error("系統被攻破了！亂碼 Token 竟然驗證成功")
		}
	})
}

func TestRefreshTokenFlow(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("啟動 miniredis 失敗: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	testKM := NewKeyManager(rdb)

	// 測試情境 1：Refresh Token 的正常核發與驗證
	t.Run("Refresh Token 正常運作", func(t *testing.T) {
		username := "Mike_Refresh_Test"
		uid := "123"
		//產生一組雙 Token
		_, refreshToken, err := GenerateToken(ctx, testKM, uid, username)
		if err != nil {
			t.Fatalf("產生 Tokens 失敗: %v", err)
		}
		//驗證 Refresh Token
		parsedClaims, err := ValidateRefreshToken(ctx, testKM, refreshToken)
		if err != nil {
			t.Fatalf("驗證 Refresh Token 失敗: %v", err)
		}
		if parsedClaims.Name != username {
			t.Errorf("Refresh Token 解碼名稱錯誤：期望 %s，得到 %s", username, parsedClaims.Name)
		}
	})
	// 測試情境 2：【重要】防禦交叉污染 (Access vs Refresh)
	// 確保 Access Token 不能拿去通過 Refresh Token 的驗證器
	t.Run("防禦 Access Token 混入 Refresh 流程", func(t *testing.T) {
		username := "Mike_Cross_Test"
		uid := "321"

		acessToken, _, _ := GenerateToken(ctx, testKM, uid, username)

		// 嘗試用 Access Token 去跑 ValidateRefreshToken
		_, err := ValidateRefreshToken(ctx, testKM, acessToken)

		// 預期失敗：因為 RefreshToken 通常會檢查不同的 secret key 或 claim
		if err == nil {
			t.Error("嚴重安全性風險！Access Token 竟然通過了 Refresh Token 的驗證器")
		}
	})

	// 測試情境 3：Refresh Token 被篡改
	t.Run("Refresh Token 被篡改", func(t *testing.T) {
		_, refreshToken, _ := GenerateToken(ctx, testKM, "999", "Hacker")
		// 隨意修改一下字串
		tampered := refreshToken + "hacked"
		_, err := ValidateRefreshToken(ctx, testKM, tampered)
		if err == nil {
			t.Error("篡改過的 Refresh Token 竟然驗證成功")
		}
	})
}
