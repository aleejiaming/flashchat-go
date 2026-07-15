// public/js/main.js
import { DOM, initEmojiPicker, appendSidebarMessage } from './ui.js';
import { connectWebSocket, sendMessage } from './socket.js';
import { registerUser, loginUser, guestLogin, getMyNickname } from './auth.js';

let currentAuthMode = 'login'; // 'login', 'register', 'guest'

window.onload = () => {
    initEmojiPicker();
    setupEventListeners();

 // 🌟 啟動時自動檢查登入狀態！
    try {
        const authData = await verifySession();
        // 如果成功換發 Token，直接跳過登入視窗進入聊天室
        enterChatRoom(authData);
        appendSidebarMessage("SYSTEM", "歡迎回來！登入狀態已自動恢復。");
    } catch (e) {
        // 如果沒有 Cookie 或 Cookie 過期，就停留在原本的登入視窗
        console.log("需要重新登入");
    }
};


function setupEventListeners() {
    // === 頁籤切換邏輯 (加上 DOM. 前綴) ===
    DOM.tabLogin.onclick = () => switchMode('login');
    DOM.tabRegister.onclick = () => switchMode('register');x``
    DOM.tabGuest.onclick = () => switchMode('guest');

    // === 送出按鈕綁定 (加上 DOM. 前綴) ===
    DOM.btnMemberSubmit.onclick = handleMemberSubmit;
    DOM.btnGuestSubmit.onclick = handleGuestSubmit;

    // 發送聊天訊息事件
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

    document.addEventListener('click', (e) => {
        if (e.target !== DOM.emojiPicker && e.target !== DOM.emojiBtn && !DOM.emojiPicker.contains(e.target)) {
            DOM.emojiPicker.style.display = 'none';
        }
    });
}

// UI 頁籤切換功能
function switchMode(mode) {
    currentAuthMode = mode;
    DOM.errorMsg.style.display = 'none'; // 切換時清空錯誤訊息
    
    // 按鈕顏色重置
    [DOM.tabLogin, DOM.tabRegister, DOM.tabGuest].forEach(btn => {
        btn.style.backgroundColor = 'var(--retro-dark)';
        btn.style.color = 'var(--retro-green)';
    });

    if (mode === 'guest') {
        DOM.tabGuest.style.backgroundColor = '#fff';
        DOM.tabGuest.style.color = '#000';
        DOM.formMember.style.display = 'none';
        DOM.formGuest.style.display = 'block';
    } else {
        const activeTab = mode === 'login' ? DOM.tabLogin : DOM.tabRegister;
        activeTab.style.backgroundColor = '#fff';
        activeTab.style.color = '#000';
        DOM.formMember.style.display = 'block';
        DOM.formGuest.style.display = 'none';
        DOM.btnMemberSubmit.innerText = mode === 'login' ? 'LOGIN' : 'REGISTER';
    }
}

// 顯示錯誤訊息的小工具
function showError(msg) {
    DOM.errorMsg.innerText = msg;
    DOM.errorMsg.style.display = 'block';
}

// 處理會員 (登入/註冊) 提交
async function handleMemberSubmit() {
    // 🌟 修正：從 DOM.inputUsername 取得資料
    const user = DOM.inputUsername.value.trim();
    const pass = DOM.inputPassword.value;

    if (!user || !pass) {
        showError("Username and Password are required!");
        return;
    }

    try {
        if (currentAuthMode === 'register') {
            await registerUser(user, pass);
            alert("註冊成功！系統將為您自動登入。");
        }

        const authData = await loginUser(user, pass);
        enterChatRoom(authData);
    } catch (error) {
        showError(error.message);
    }
}

// 處理遊客提交
async function handleGuestSubmit() {
    // 🌟 修正：從 DOM.inputNickname 取得資料
    const nick = DOM.inputNickname.value.trim();
    try {
        const authData = await guestLogin(nick);
        enterChatRoom(authData);
    } catch (error) {
        showError(error.message);
    }
}

// 成功取得 Token 後，進入聊天室的通用邏輯
function enterChatRoom(authData) {
    DOM.displayName.innerText = authData.username + ">";
    DOM.loginModal.style.display = 'none';
    DOM.messageInput.focus();
    
    // 啟動 WebSocket 連線
    connectWebSocket(authData.token);
}

// 發送訊息邏輯
function handleSend() {
    const text = DOM.messageInput.value.trim();
    if (text === "") return;

    sendMessage({
        username: getMyNickname(),
        content: text
    });

    DOM.messageInput.value = "";
}