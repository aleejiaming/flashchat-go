package ws

import (
	"reflect" // Go 內建的反射套件，可以用來偷看變數的「真實型態」
	"testing" // Go 內建的測試套件
)

// 🌟 規則 1：測試函式的名稱，必須以大寫的 Test 開頭
// 🌟 規則 2：必須傳入 t *testing.T，這是讓你能對考卷打分數的紅筆

func TestGetProcessor(t *testing.T) {
	tests := []struct {
		name     string // 測試名稱
		input    string // 客人輸入的文字
		expected string // 預期會派出的機器人型態
	}{
		{"幫助指令", "/help", "*ws.HelpProcessor"},
		{"私聊指令", "/msg Mike 嗨", "*ws.PrivateMsgProcessor"},
		{"天氣指令", "/weather 台北", "*ws.WeatherProcessor"},
		{"一般聊天", "今天天氣真好", "*ws.NormalProcessor"},
	}

	// 2. 執行測試邏輯 (Logic)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := GetProcessor(tt.input)
			actual := reflect.TypeOf(p).String()

			if actual != tt.expected {
				t.Errorf("期望拿到 %s，卻拿到 %s", tt.expected, actual)
			}
		})
	}
}

// ==========================================
// 📝 考卷二：測試「所有主廚做菜邏輯」 (表格驅動測試)
// ==========================================

func TestMessageProcessor(t *testing.T) {
	// 1. 定義測試資料表
	tests := []struct {
		name          string  // 測試名稱
		inputMsg      Message // 客人傳來的原始訊息
		expectPrivate bool    // 預期是否為私訊
		expectTarget  string  // 預期的收件人 (沒有就留空)
		expectName    string  // 預期的發件人 (用來檢查是否被改成機器人)
	}{
		{"Help 測試", Message{Content: "/help"}, true, "", "🤖 系統機器人"},
		{"Msg 正確格式", Message{Content: "/msg Mike 打電動？"}, true, "Mike", ""},
		{"Msg 錯誤格式", Message{Content: "/msg"}, true, "", "🤖 系統機器人"},
		{"一般聊天", Message{Name: "Jack", Content: "大家好"}, false, "", "Jack"},
	}

	// 2. 執行測試邏輯
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先請工廠派人，再請機器人處理訊息
			p := GetProcessor(tt.inputMsg.Content)
			result := p.Process(tt.inputMsg)

			// 驗證三個核心結果
			if result.IsPrivate != tt.expectPrivate {
				t.Errorf("IsPrivate 期望 %v, 實際 %v", tt.expectPrivate, result.IsPrivate)
			}
			if tt.expectTarget != "" && result.TargetName != tt.expectTarget {
				t.Errorf("收件人 期望 '%s', 實際 '%s'", tt.expectTarget, result.TargetName)
			}
			if tt.expectName != "" && result.Name != tt.expectName {
				t.Errorf("發件人 期望 '%s', 實際 '%s'", tt.expectName, result.Name)
			}
		})
	}
}
