package ws

import (
	"strings" // 🌟 記得引入 strings，因為我們的工廠需要用到 HasPrefix 比對字串

)
// 2. 機器人處理器 (合約)
type MessageProcessor interface {
	Process(msg Message) Message
}

// -- 一般聊天處理器 --
type NormalProcessor struct{}
func (p *NormalProcessor) Process(msg Message) Message { return msg }

// -- 幫助選單處理器 --
type HelpProcessor struct{}
func (p *HelpProcessor) Process(msg Message) Message {
	msg.Name = "🤖 系統機器人"
	msg.Content = "目前支援的指令：\n/weather [城市] - 查詢天氣\n/help - 顯示此選單"
	return msg
}

// -- 天氣預報處理器 --
type WeatherProcessor struct{}
func (p *WeatherProcessor) Process(msg Message) Message {
	msg.Name = "🌤️ 天氣機器人"
	msg.Content = "今天天氣晴朗，氣溫 28 度！(模擬資料)"
	return msg
}

// 3. 終極工廠：決定派誰出場
func GetProcessor(content string) MessageProcessor {
	if strings.HasPrefix(content, "/weather") {
		return &WeatherProcessor{}
	} else if strings.HasPrefix(content, "/help") {
		return &HelpProcessor{}
	}
	return &NormalProcessor{}
}