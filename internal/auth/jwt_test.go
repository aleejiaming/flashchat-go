package auth

import (
	"strings"
	"testing"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	// 測試情境 1：正常的 Token 簽發與解析 (Round-trip Test)
	t.Run("正常核發與驗證", func(t *testing.T) {
		testUser := "Mike_Testing"

		// 1. 產生 Token
		token, _, err := GenerateToken(testUser)
		if err != nil {
			t.Fatalf("產生 Token 失敗: %v", err)
		}

		if token == "" {
			t.Error("預期拿到 token 字串，卻拿到空字串")
		}

		// 2. 驗證剛剛產生的 Token
		parsedUser, err := ValidateToken(token)
		if err != nil {
			t.Fatalf("驗證 Token 失敗: %v", err)
		}

		// 3. 檢查解碼出來的名字跟原本是不是一樣
		if parsedUser != testUser {
			t.Errorf("解碼名稱錯誤：期望得到 %s，卻得到 %s", testUser, parsedUser)
		}
	})

	// 測試情境 2：防禦偽造或篡改的 Token
	t.Run("防禦篡改的Token", func(t *testing.T) {
		testUser := "Hacker"
		token, _, _ := GenerateToken(testUser)

		// 駭客嘗試破壞/竄改 token 的內容 (隨便把一段字串轉成大寫)
		tamperedToken := strings.ToUpper(token)

		// 驗證這個被改過的 Token
		_, err := ValidateToken(tamperedToken)

		// 我們「期望」這裡必須要有 error (因為這是不合法的 Token)
		if err == nil {
			t.Error("系統被攻破了！篡改過的 Token 竟然驗證成功")
		}
	})

	// 測試情境 3：完全無效的亂碼 Token
	t.Run("無效格式防呆", func(t *testing.T) {
		fakeToken := "this.is.not.a.real.jwt.token"

		_, err := ValidateToken(fakeToken)

		if err == nil {
			t.Error("系統被攻破了！亂碼 Token 竟然驗證成功")
		}
	})
}

func TestRefreshTokenFlow(t *testing.T) {
	// 測試情境 1：Refresh Token 的正常核發與驗證
	t.Run("Refresh Token 正常運作", func(t *testing.T) {
		username := "Mike_Refresh_Test"

		//產生一組雙 Token
		_, refreshToken, err := GenerateToken(username)
		if err != nil {
			t.Fatalf("產生 Tokens 失敗: %v", err)
		}
		//驗證 Refresh Token
		parsedUser, err := ValidateRefreshToken(refreshToken)
		if err != nil {
			t.Fatalf("驗證 Refresh Token 失敗: %v", err)
		}
		if parsedUser != username {
			t.Errorf("Refresh Token 解碼名稱錯誤：期望 %s，得到 %s", username, parsedUser)
		}
	})
	// 測試情境 2：【重要】防禦交叉污染 (Access vs Refresh)
	// 確保 Access Token 不能拿去通過 Refresh Token 的驗證器
	t.Run("防禦 Access Token 混入 Refresh 流程", func(t *testing.T) {
		username := "Mike_Cross_Test"
		acessToken, _, _ := GenerateToken(username)

		// 嘗試用 Access Token 去跑 ValidateRefreshToken
		_, err := ValidateRefreshToken(acessToken)

		// 預期失敗：因為 RefreshToken 通常會檢查不同的 secret key 或 claim
		if err == nil {
			t.Error("嚴重安全性風險！Access Token 竟然通過了 Refresh Token 的驗證器")
		}
	})

	// 測試情境 3：Refresh Token 被篡改
	t.Run("Refresh Token 被篡改", func(t *testing.T) {
		_, refreshToken, _ := GenerateToken("Hacker")
		// 隨意修改一下字串
		tampered := refreshToken + "hacked"
		_, err := ValidateRefreshToken(tampered)
		if err == nil {
			t.Error("篡改過的 Refresh Token 竟然驗證成功")
		}
	})
}
