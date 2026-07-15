// public/js/auth.js

let currentNickname = "Guest";
let currentToken = "";

// 註冊邏輯
export async function registerUser(username, password) {
    const response = await fetch('/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
    });
    
    if (!response.ok) {
        const errMsg = await response.text();
        throw new Error(errMsg.trim() || "註冊失敗");
    }
    return true; // 註冊成功
}

// 會員登入邏輯
export async function loginUser(username, password) {
    const response = await fetch('/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
    });

    if (!response.ok) throw new Error("帳號或密碼錯誤");

    const data = await response.json();
    currentToken = data.token;
    currentNickname = data.username;
    // 注意：Refresh Token 已經自動被瀏覽器存入 Cookie 了，我們不需要處理它
    return { token: currentToken, username: currentNickname };
}

// 遊客登入邏輯
export async function guestLogin(nickname) {
    const response = await fetch('/guest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: nickname }) // 後端 struct 一律吃 username
    });

    if (!response.ok) throw new Error("遊客登入失敗");

    const data = await response.json();
    currentToken = data.token;
    currentNickname = data.username;
    return { token: currentToken, username: currentNickname };
}

// 🌟 新增：自動恢復連線 (Auto Login via Refresh Token)
export async function verifySession() {
    // 呼叫 /refresh，瀏覽器會自動帶上 HttpOnly Cookie
    const response = await fetch('/refresh', { method: 'POST' });
    
    if (!response.ok) {
        throw new Error("無效的登入狀態或憑證已過期");
    }

    const data = await response.json();
    currentToken = data.token;
    currentNickname = data.username;
    return { token: currentToken, username: currentNickname };
}

// 🌟 新增：登出
export async function logoutUser() {
    await fetch('/logout', { method: 'POST' });
    currentToken = "";
    currentNickname = "Guest";
}

export function getMyNickname() { return currentNickname; }
export function getToken() { return currentToken; }