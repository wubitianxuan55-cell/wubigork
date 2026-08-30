// Package weixin — iLink 媒体 AES-128-ECB 加解密（v4.8.3 真机协议定稿）。
//
// 协议证据（三方印证）：
//   - 本机抓包实测：CDN 密文 AES-128-ECB(hexdecode(image_item.aeskey), PKCS7)
//     解出 JPEG（hd_size 与明文长度精确一致，首块 FF D8 FF E0 JFIF）
//   - hermes-agent weixin.py 生产实现（NousResearch）同款：_aes128_ecb_encrypt/
//     _aes128_ecb_decrypt + pkcs7，aes_key 走 base64(hex 字符串)
//   - openilink-sdk-go 导出常量 ENCRYPT_AES128_ECB
//
// Go 标准库无 ECB 模式，这里用 cipher.Block 逐块实现（ECB 无 IV，各块独立）。
package weixin

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const aesBlockSize = aes.BlockSize // 16

// ecbEncrypter / ecbDecrypter 把 cipher.Block 包装成 ECB 模式（无 IV，
// 逐块独立加解密；输入长度必须是块大小的整数倍）。
type ecbBlockMode struct {
	b         cipher.Block
	decryptin bool
}

func (m ecbBlockMode) BlockSize() int { return m.b.BlockSize() }

func (m ecbBlockMode) CryptBlocks(dst, src []byte) {
	if len(src)%m.b.BlockSize() != 0 {
		panic("crypto/cipher: input not full blocks (ECB)")
	}
	for i := 0; i < len(src); i += m.b.BlockSize() {
		if m.decryptin {
			m.b.Decrypt(dst[i:], src[i:])
		} else {
			m.b.Encrypt(dst[i:], src[i:])
		}
	}
}

// aes128ECBEncrypt AES-128-ECB 加密（明文先 PKCS7 对齐），key 必须 16 字节。
func aes128ECBEncrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("AES-128 密钥须为 16 字节, 实际 %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plaintext, aesBlockSize)
	out := make([]byte, len(padded))
	ecbBlockMode{b: block}.CryptBlocks(out, padded)
	return out, nil
}

// aes128ECBDecrypt AES-128-ECB 解密（PKCS7 去填充；末字节非合法填充值时
// 原样返回去填充前数据——宽容处理真实 CDN 偶发的尾随字节，不报错）。
func aes128ECBDecrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("AES-128 密钥须为 16 字节, 实际 %d", len(key))
	}
	if len(ciphertext) == 0 || len(ciphertext)%aesBlockSize != 0 {
		return nil, fmt.Errorf("密文长度 %d 非块大小 %d 的整数倍", len(ciphertext), aesBlockSize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(ciphertext))
	ecbBlockMode{b: block, decryptin: true}.CryptBlocks(out, ciphertext)
	return pkcs7Unpad(out, aesBlockSize), nil
}

// pkcs7Pad PKCS7 对齐：恒追加 1..block 字节填充（明文已对齐时整块填充）。
func pkcs7Pad(data []byte, block int) []byte {
	pad := block - len(data)%block
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

// pkcs7Unpad PKCS7 去填充（hermes 同款宽容语义：填充不合法时返回原文，
// 宁可交给上层魔数终审，不在这里误杀）。
func pkcs7Unpad(data []byte, block int) []byte {
	if len(data) == 0 || len(data)%block != 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > block {
		return data
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return data
		}
	}
	return data[:len(data)-pad]
}

// parseAESKeyHex 解析 iLink 媒体密钥：32 位 hex（→16 字节）。兼容原始
// 16 字节的 base64 形态（media.aes_key = base64(hex字符串)，解析链见
// imageItem.resolveDownload）。
func parseAESKeyHex(hexKey string) ([]byte, error) {
	raw, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("aeskey 非合法 hex: %w", err)
	}
	if len(raw) != 16 {
		return nil, fmt.Errorf("aeskey 解码后 %d 字节, 期望 16", len(raw))
	}
	return raw, nil
}

// aesKeyForAPI 出站消息的 media.aes_key 编码：**base64(hex 字符串)**，不是
// base64(原始字节)——hermes 实测注释：发 base64(原始字节) 会让接收端解出灰框。
func aesKeyForAPI(key []byte) string {
	return base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(key)))
}

// aesKeyFromBase64Hex 还原 media.aes_key（base64(hex 字符串)）为原始字节。
func aesKeyFromBase64Hex(s string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("aes_key base64 解码失败: %w", err)
	}
	return parseAESKeyHex(string(decoded))
}

// md5Hex 文件内容 MD5（getuploadurl 的 rawfilemd5 字段）。
func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}
