// public/js/main.js

//這是前端程式的進入點，負責將 UI、Socket 與驗證邏輯串接起來，並綁定事件監聽器。
import { DOM, initEmojiPicker } from './ui.js';
import { connectWebSocket, sendMessage } from './socket.js';
import { processLogin, getMyNickname } from './auth.js';

// 初始化
window.onload = () => {
    initEmojiPicker();
    setupEventListeners();
};

// 集中管理所有的事件綁定 (Event Listeners)
function setupEventListeners() {
    // 登入事件
    DOM.joinBtn.addEventListener('click', handleLogin);
    DOM.nicknameInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') handleLogin();
    });

    // 發送訊息事件
    DOM.sendBtn.addEventListener('click', handleSend);
    DOM.messageInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') handleSend();
    });

    // 表情符號面板開關
    DOM.emojiBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const picker = DOM.emojiPicker;
        picker.style.display = picker.style.display === 'grid' ? 'none' : 'grid';
    });

    // 點擊空白處關閉表情面板
    document.addEventListener('click', (e) => {
        if (e.target !== DOM.emojiPicker && e.target !== DOM.emojiBtn && !DOM.emojiPicker.contains(e.target)) {
            DOM.emojiPicker.style.display = 'none';
        }
    });
}

// 處理登入邏輯
async function handleLogin() {
    const rawName = DOM.nicknameInput.value;
   // 等待 API 回傳 JWT Token
   // const authData = await processLogin(rawName);
    // 🌟 修復 2：加上 await 等待 API 回傳結果
    const authData = await processLogin(rawName);

    // 防呆機制：如果登入 API 發生錯誤 (例如後端沒開)，就中斷執行
    if (!authData) {
        alert("登入失敗，請檢查系統連線");
        return;
    }

    // 🌟 修復 3：正確使用 authData 裡面的屬性
    DOM.displayName.innerText = authData.finalName + ">";
    DOM.loginModal.style.display = 'none';
    DOM.messageInput.focus();
    
    // 🌟 修復 4：正確傳遞 token
    connectWebSocket(authData.token);
}

// 處理發送訊息邏輯
function handleSend() {
    const text = DOM.messageInput.value.trim();
    if (text === "") return;

    sendMessage({
        username: getMyNickname(),
        content: text
    });

    DOM.messageInput.value = "";
}