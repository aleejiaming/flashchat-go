// public/js/ui.js

// 取得所有需要的 DOM 元素並導出供其他模組使用
export const DOM = {
    loginModal: document.getElementById('login-modal'),
    nicknameInput: document.getElementById('nickname-input'),
    joinBtn: document.getElementById('join-btn'),
    displayName: document.getElementById('display-name'),
    messageInput: document.getElementById('message-input'),
    sendBtn: document.getElementById('send-btn'),
    emojiBtn: document.getElementById('emoji-btn'),
    emojiPicker: document.getElementById('emoji-picker'),
    chatSidebar: document.getElementById('chat-sidebar'),
    danmakuZone: document.getElementById('danmaku-zone'),

    tabLogin : document.getElementById('tab-login'),
    tabRegister : document.getElementById('tab-register'),
    tabGuest : document.getElementById('tab-guest'),
    formMember : document.getElementById('form-member'),
    formGuest : document.getElementById('form-guest'),
    btnMemberSubmit : document.getElementById('submit-member-btn'),
    btnGuestSubmit : document.getElementById('submit-guest-btn'),
    inputUsername : document.getElementById('username-input'),
    inputPassword : document.getElementById('password-input'),
    inputNickname : document.getElementById('nickname-input'),
    errorMsg : document.getElementById('auth-error-msg'),
};

// 渲染側邊欄訊息
export function appendSidebarMessage(time, name, msg) {
    const msgDiv = document.createElement('div');
    msgDiv.className = 'chat-msg';
    
    let color = "var(--retro-green)";
    if (msg && msg.startsWith('/')) {
        color = "#ff00ff"; // 指令高亮
    }

    msgDiv.innerHTML = `<span class="chat-time">[${time}]</span> <span class="chat-name">${name}:</span> <span style="color:${color}">${msg}</span>`;
    DOM.chatSidebar.appendChild(msgDiv);
    DOM.chatSidebar.scrollTop = DOM.chatSidebar.scrollHeight;
}

// 發射彈幕
export function fireDanmaku(msg) {
    const danmaku = document.createElement('div');
    danmaku.className = 'danmaku-item';
    danmaku.innerText = msg;

    const randomTop = Math.floor(Math.random() * 80) + 10;
    danmaku.style.top = `${randomTop}%`;

    const randomSpeed = Math.floor(Math.random() * 4) + 6;
    danmaku.style.animationDuration = `${randomSpeed}s`;

    DOM.danmakuZone.appendChild(danmaku);

    setTimeout(() => {
        danmaku.remove();
    }, randomSpeed * 1000);
}

// 初始化表情符號面板
export function initEmojiPicker() {
    const emojis = ['😀', '😂', '😍', '😎', '😭', '😡', '👍', '🙏', '🚀', '🔥', '🐛', '👽', '👻', '🍕', '🎮'];
    emojis.forEach(emoji => {
        const span = document.createElement('span');
        span.className = 'emoji-item';
        span.innerText = emoji;
        span.onclick = () => {
            DOM.messageInput.value += emoji;
            DOM.messageInput.focus();
            DOM.emojiPicker.style.display = 'none';
        };
        DOM.emojiPicker.appendChild(span);
    });
}