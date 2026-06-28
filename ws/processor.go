package ws

import (
	"strings" // 🌟 記得引入 strings，因為我們的工廠需要用到 HasPrefix 比對字串
	"net/http"
	"io"

)

// ==========================================
// 1. 定義通訊格式 (🌟 新增 IsPrivate 標籤)
// ==========================================
type Message struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	IsPrivate bool   `json:"-"` // 加上 json:"-" 代表這只是後端內部用的，不會傳給前端
	TargetName string `json:"-"` // 🌟 新增：這封信的「指定收件人」是誰？
}

// ==========================================
// 2. 機器人處理器 (合約)
// ==========================================
type MessageProcessor interface {
	Process(msg Message) Message
}

// ------------------------------------------
// 💬 一般聊天處理器
// ------------------------------------------
type NormalProcessor struct{}

func (p *NormalProcessor) Process(msg Message) Message {
	// 一般聊天不用私訊，所以維持預設的 IsPrivate = false
	return msg
}

// -- 幫助選單處理器 --
type HelpProcessor struct{}
func (p *HelpProcessor) Process(msg Message) Message {
	msg.Name = "🤖 系統機器人"
	msg.Content = "目前支援的指令：\n/weather [城市] - 查詢天氣\n/help - 顯示此選單"
	msg.IsPrivate = "true"  // 🌟 魔法：告訴經理(Hub)這是悄悄話！
	return msg
}

// -- 天氣預報處理器 --
type WeatherProcessor struct{}
func (p *WeatherProcessor) Process(msg Message) Message {
	// 1. 解析城市名稱 (從 "/weather 台北" 中切出 "台北")
	parts:= strings.splitN(msg.content," ",2)
	city := "台北" //預設城市
	if len(parts) == 2 && string.TrimSpace(parts[1]) !={
		city = strings.TrimSpace(parts[1])
	}

	msg.Name = "🌤️ 天氣機器人"

	apiURL = "https://wttr.in/" + city + "?format=3"
	resp, err = http.GET(apiURL) 
	if err != nil{
		msg.content = "抱歉，氣象局連線異常 📡"
		return msg
	}
	// 🌟 關鍵習慣：用完網路連線一定要關閉，否則會造成記憶體洩漏！
	defer resp.Body.Close()

	// 3. 讀取 API 回傳的資料 (Body)
	body, err := io.ReadAll(resp.body) //這裡選用 ReadAll 是方便實作 不考量大資料 讀寫導致的 OOM kill
	if err != nil{
		msg.content = "抱歉，無法解讀天氣資料 🌧️"
		return msg
	}
	
	// 4. 將拿到的真實天氣資料，轉換為字串並回傳
	weatherInfo := string(body)
	if strings.Contains(weatherInfo,"Unknown"){
		msg.content = "找不到「" + city + "」的天氣，請試試看英文拼音喔！"
	}else{
		msg.content = "為您播報即時天氣：\n" + weatherInfo
	}
	return mse
}

// ------------------------------------------
// 🤫 私聊頻道處理器 (🌟 精確修改位置：全新加入的主廚)
// ------------------------------------------
type PrivateMsgProcessor struct{}

type (p *PrivateMsgProcessor) Process(msg Message) Message{
// 假設客人的輸入是： "/msg Mike 借我一百塊"
	// 我們用空白切成 3 份：["/msg", "Mike", "借我一百塊"]
	parts := strings.SplitN(msg.Content, " ", 3)

// 如果客人打錯格式 (例如只打了 "/msg Mike")
	if len(parts) < 3 {
		msg.Name = "🤖 系統機器人"
		msg.Content = "私訊格式錯誤！請輸入：/msg [對象名字] [你想說的話]"
		msg.IsPrivate = true //只針對發話者 的選項
		return msg
	}
	//抽出目標名子與訊息
	targe = patrs[1]
	realContent = parts[2]

	// 🌟 貼上私訊標籤，並寫上收件人是誰！
	msg.IsPrivate = true
	msg.TargetName = targer
	msg.Conn = "悄悄話"+ realContent
	
	return msg
}

// 3. 終極工廠：決定派誰出場
func GetProcessor(content string) MessageProcessor {
	if strings.HasPrefix(content, "/weather") {
		return &WeatherProcessor{}
	} else if strings.HasPrefix(content, "/help") {
		return &HelpProcessor{}
	}else if strings.HasPrefix(content, "/msg") { // 🌟 註冊私訊主廚！
		return &PrivateMsgProcessor{}
	}
	return &NormalProcessor{}
}