package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
)

type KeyManager struct {
	rdb *redis.Client
}

// 聘請倉管員 (把 Redis 鑰匙交給他)

func NewKeyManager(rdb *redis.Client) *KeyManager {
	slog.Info("密鑰管理已就緒", "component", "key_manager")
	return &KeyManager{rdb: rdb}
}

// ==========================================
// 2. 內部工具：產生高強度隨機密鑰
// ==========================================
func generateRandomSecret() string {
	bytes := make([]byte, 32) // 產生 256 bits 的隨機位元組
	_, err := rand.Read(bytes)
	if err != nil {
		slog.Error("無法產生隨機密鑰", "component", "key_manager", "error", err.Error())
		panic("無法產生隨機密鑰:" + err.Error())
	}
	return hex.EncodeToString(bytes) //轉成 16 進位數字
}

// ==========================================
// 3. 核心功能：執行密鑰輪換 (Rotate)
// ==========================================

func (km *KeyManager) RotateKey(ctx context.Context) error {
	newKID := fmt.Sprintf("key_%d", time.Now().Unix())
	newSecret := generateRandomSecret()

	// 🌟 記錄開始輪換的動作與新產生的 KID (使用 Debug 層級，避免正式環境洗頻，但開發時很好用)
	slog.Debug("開始執行密鑰輪換作業...", "component", "key_manager", "new_kid", newKID)

	// 將新鑰匙存入 active_keys
	// 🌟 使用 map 明確指定 Key-Value，避免套件解析錯誤或人為寫反
	err := km.rdb.HSet(ctx, "active_keys", map[string]interface{}{
		newKID: newSecret,
	}).Err()
	if err != nil {
		// 🌟 記錄寫入 Redis 失敗的詳細資訊
		slog.Error("❌ 寫入 active_keys 失敗", "component", "key_manager", "error", err.Error(), "kid", newKID)
		return fmt.Errorf("寫入 active_keys 失敗: %v", err)
	}

	// 更新 primary_kid 告示牌
	err = km.rdb.Set(ctx, "primary_kid", newKID, 0).Err()
	if err != nil {
		// 🌟 記錄更新指標失敗的詳細資訊
		slog.Error("❌ 更新 primary_kid 失敗", "component", "key_manager", "error", err.Error(), "kid", newKID)
		return fmt.Errorf("更新 primary_kid 失敗: %v", err)
	}

	// 🌟 成功完成輪換，留下清楚的成功紀錄
	slog.Info("✅ 密鑰輪換完成", "component", "key_manager", "active_primary_kid", newKID)

	return nil
}

// ==========================================
// 4. 取得目前簽發用的主鑰匙 (Primary Key)
// ==========================================
// 用途：當使用者登入，我們需要簽發全新 JWT 時呼叫。
func (km *KeyManager) GetPrimarySecret(ctx context.Context) (string, string, error) {
	// 先看公佈欄，查出 primary_kid 是誰
	kid, err := km.rdb.Get(ctx, "primary_kid").Result()

	// 🌟 新增防呆機制：如果 Redis 說 "Nil" (找不到)
	if err == redis.Nil {
		slog.Info("⚠️ 偵測到系統尚未初始化密鑰，立刻觸發首次打磨...")

		// 當場請倉管員打磨第一把鑰匙
		if rotateErr := km.RotateKey(ctx); rotateErr != nil {
			return "", "", fmt.Errorf("自動初始化密鑰失敗: %v", rotateErr)
		}
		// 打磨完後，重新看一次公佈欄拿 kid
		kid, err = km.rdb.Get(ctx, "primary_kid").Result()
	} else if err != nil {
		return "", "", fmt.Errorf("尋找 primary_kid 發生錯誤: %v", err)
	}

	// 2. 去 active_keys 抽屜拿出對應的密鑰字串
	secret, err := km.rdb.HGet(ctx, "active_keys", kid).Result()

	// 🌟 自癒機制 (Auto-Healing)：資料不一致時自動修復

	if err == redis.Nil || secret == "" {
		slog.Warn("🚨 發現公佈欄的密鑰遺失 (資料不一致)！啟動自癒機制，重新打磨密鑰...", "corrupted_kid", kid)

		//充新打磨密鑰覆蓋髒資料
		if rotateErr := km.RotateKey(ctx); rotateErr != nil {
			return "", "", fmt.Errorf("自癒打磨密鑰失敗: %v", rotateErr)
		}

		// 重新從公佈欄與抽屜拿取最新資料
		kid, _ = km.rdb.Get(ctx, "primary_kid").Result()
		secret, err = km.rdb.HGet(ctx, "active_keys", kid).Result()
	}

	// 如果經歷過自癒還是拿不到，才是真的系統掛了
	if err != nil {
		slog.Error("❌ 無法取得主密鑰", "kid", kid, "error", err.Error())
		return "", "", fmt.Errorf("找不到主密鑰 (%s): %v", kid, err)
	}

	return kid, secret, nil
}

// ==========================================
// 5. 透過 kid 取得對應的密鑰 (用於驗證舊 Token)
// ==========================================
// 用途：當使用者拿著 JWT 來敲門，我們看他 Header 上的 kid，來這裡找對應的鑰匙驗證。

func (km *KeyManager) GetSecretByKeyID(ctx context.Context, kid string) (string, error) {
	secret, err := km.rdb.HGet(ctx, "active_keys", kid).Result()
	if err != nil {
		// 如果在抽屜裡找不到，代表這把鑰匙已經過期被系統淘汰，或者是偽造的
		slog.Warn("⚠️ 嘗試使用無效或已淘汰的密鑰", "component", "key_manager", "kid", kid)
		return "", fmt.Errorf("無效或已淘汰的 kid: %s", kid)
	}
	return secret, nil
}

// 🌟 新增：檢查 UID 是否在 Redis 黑名單中
func (km *KeyManager) IsUserBlacklisted(ctx context.Context, uid string) (bool, error) {
	if uid == "" {
		return false, nil
	}
	// Key 格式範例：blacklist:user:1001
	key := fmt.Sprintf("blacklist:user:%s", uid)

	exists, err := km.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, err
}

// 🌟 新增：管理員使用的封鎖 API 邏輯 (可設定封鎖時長)
func (km *KeyManager) BanUser(ctx context.Context, uid string, duration time.Duration) error {
	key := fmt.Sprintf("blacklist:user:%s", uid)
	// 在 Redis 中記錄 banned，並設定 TTL 過期時間 (若 duration 為 0 則代表永久封鎖)
	return km.rdb.Set(ctx, key, "banned", duration).Err()
}
