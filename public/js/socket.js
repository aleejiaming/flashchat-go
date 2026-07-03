// public/js/socket.js
import { appendSidebarMessage, fireDanmaku } from './ui.js';
import { getMyNickname } from './auth.js';

let socket = null;
let heartbeatTimer = null;

export function connectWebSocket(token) {
    // 實作帶有 Token 的連線
    const wsUrl = `ws://${window.location.host}/ws?token=${token}`;
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
        appendSidebarMessage("SYSTEM", "Connected to FlashChat server.");
        
        // 啟動心跳排程 (每 30 秒)
        heartbeatTimer = setInterval(() => {
            if (socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({
                    name: getMyNickname(),
                    content: "/ping"
                }));
            }
        }, 30000);
    };

    socket.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        
        // 攔截系統心跳回應
        if (msg.content === "pong") return; 

        const timeStr = new Date().toTimeString().split(' ')[0];
        appendSidebarMessage(timeStr, msg.username, msg.content);
        fireDanmaku(msg.content);
    };

    socket.onclose = () => {
        appendSidebarMessage("SYSTEM", "Connection lost. Reconnecting in 3s...");
        clearInterval(heartbeatTimer);
        setTimeout(() => connectWebSocket(token), 3000);
    };
}

export function sendMessage(msgObj) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(msgObj));
    } else {
        appendSidebarMessage("SYSTEM", "Not connected to server!");
    }
}