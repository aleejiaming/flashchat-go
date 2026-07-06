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

export function getMyNickname() { return currentNickname; }
export function getToken() { return currentToken; }