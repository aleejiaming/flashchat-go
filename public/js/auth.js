// public/js/auth.js

// 模組內部狀態 (State)
let currentNickname = "Guest";
let currentToken = "";

/**
 * 處理登入邏輯並發送 POST 請求取得 JWT
 */
export async function processLogin(rawName) {
    currentNickname = rawName.trim() || "Guest";
    
    try {
        const response = await fetch('/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ name: currentNickname })
        });

        if (!response.ok) {
            throw new Error(`HTTP 錯誤狀態: ${response.status}`);
        }

        const data = await response.json();
        currentToken = data.token; // 取得後端簽發的真實 JWT

        return {
            finalName: currentNickname,
            token: currentToken
        };
    // 💡 錯誤通常發生在這裡：忘記補上 catch 區塊與大括號
    } catch (error) {
        console.error("登入請求失敗:", error);
        return null;
    }
}

/**
 * 取得當前使用者暱稱
 */
export function getMyNickname() {
    return currentNickname;
}

/**
 * 取得當前連線 Token
 */
export function getToken() {
    return currentToken;
}