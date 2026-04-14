package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// 钉钉 PC V3 公开逆向结论（与 V2 最终形态一致：16 个 ASCII 十六进制字符作为 AES-128 密钥字节）：
// 1) password = uid 字符串 + salt 字符串（来自 user_config Base64 解码后的 JSON）
// 2) dk = PBKDF2-HMAC-SHA1(password, salt="666DingTalk888" 的前 8 字节, iter=1000, keyLen=32)
// 3) key = MD5(dk) 的十六进制字符串的前 16 个字符，按字节作为密钥（与 GenerateKey 的 V2 写法一致）

const pbkdf2Pepper = "666DingTalk888"

// GenerateKeyV3 根据 V3 规则从「业务 uid」与「user_config 里的 salt」派生数据库密钥。
func GenerateKeyV3(uid, salt string) []byte {
	password := uid + salt
	pepper := []byte(pbkdf2Pepper)
	if len(pepper) > 8 {
		pepper = pepper[:8]
	}
	dk := pbkdf2.Key([]byte(password), pepper, 1000, 32, sha1.New)
	sum := md5.Sum(dk)
	hexHash := hex.EncodeToString(sum[:])
	return []byte(hexHash[:16])
}

// SaltFromUserConfigFile 读取钉钉用户目录下的 user_config（整文件 Base64），解析 JSON 取 salt/slt。
func SaltFromUserConfigFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(raw))
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("user_config Base64 解码失败: %w", err)
	}
	return saltFromUserConfigJSON(dec)
}

func saltFromUserConfigJSON(decoded []byte) (string, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(decoded, &m); err != nil {
		return "", fmt.Errorf("user_config JSON 解析失败: %w", err)
	}
	for _, key := range []string{"salt", "slt", "Salt", "Slt"} {
		raw, ok := m[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil && s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("JSON 中未找到非空的 salt/slt 字段")
}
