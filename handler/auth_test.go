package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"flashchat-go/repository"
)

// ==========================================
// 1. 建立「假倉管員」(Mock Repository)
// ==========================================
type mockUserRepository struct {
	// 用一個簡單的 map 來假裝成 PostgreSQL 資料庫
	fakeDB map[string]repository.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		fakeDB: make(map[string]repository.User),
	}
}

// 假倉管員實作 CreateUser 合約
func (m *mockUserRepository) CreateUser(username, passwordHash string) error {
	// 模擬 PostgreSQL 的 UNIQUE 限制：如果帳號已經存在，就報錯
	if _, exists := m.fakeDB[username]; exists {
		return errors.New("unique constraint violation")
	}

	m.fakeDB[username] = repository.User{
		Username:     username,
		PasswordHash: passwordHash,
	}
	return nil
}

// 假倉管員實作 GetUserByUsername 合約
func (m *mockUserRepository) GetUserByUsername(username string) (*repository.User, error) {
	user, exists := m.fakeDB[username]
	if !exists {
		return nil, errors.New("user_not_found")
	}
	return &user, nil
}

// ==========================================
// 2. 撰寫 API 單元測試
// ==========================================
func TestRegisterHandler(t *testing.T) {
	// 1. 準備環境：聘請假倉管員，並把配發給 AuthHandler
	mockRepo := newMockUserRepository()
	authHandler := NewAuthHandler(mockRepo)

	// 2. 準備測試資料表 (Table-Driven Tests)
	tests := []struct {
		name           string // 測試情境名稱
		requestBody    AuthRequest
		expectedStatus int    // 預期的 HTTP 狀態碼
		expectedBody   string // 預期回傳的文字
	}{
		{
			name:           "正常註冊",
			requestBody:    AuthRequest{Username: "testuser", Password: "password123"},
			expectedStatus: http.StatusCreated, // 201
			expectedBody:   "註冊成功",
		},
		{
			name:           "重複註冊",
			requestBody:    AuthRequest{Username: "testuser", Password: "password123"}, // 刻意用一樣的帳號
			expectedStatus: http.StatusConflict,                                        // 409
			expectedBody:   "此帳號已被註冊",
		},
		{
			name:           "空白帳號防呆",
			requestBody:    AuthRequest{Username: "   ", Password: "123"},
			expectedStatus: http.StatusBadRequest, // 400
			expectedBody:   "帳號密碼不得為空",
		},
	}

	// 3. 開始跑迴圈測試
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 將我們的測試資料打包成 JSON
			bodyBytes, _ := json.Marshal(tt.requestBody)

			// 🌟 神器登場：假裝發出一個 HTTP POST 請求到 /register
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))

			// 🌟 神器登場：準備一個「錄音機 (Recorder)」來接住 Handler 的回傳結果
			rr := httptest.NewRecorder()

			// 呼叫我們要測試的函式！
			authHandler.RegisterHandler(rr, req)

			// 驗證結果 1：檢查 HTTP 狀態碼是否如我們預期
			if rr.Code != tt.expectedStatus {
				t.Errorf("情境 [%s] 狀態碼錯誤：期望得到 %d，卻得到 %d", tt.name, tt.expectedStatus, rr.Code)
			}

			// 驗證結果 2：檢查回傳的文字內容是否有包含關鍵字
			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("情境 [%s] 回傳內容錯誤：期望包含 '%s'，實際拿到 '%s'", tt.name, tt.expectedBody, rr.Body.String())
			}
		})
	}
}
