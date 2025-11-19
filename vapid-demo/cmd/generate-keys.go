package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	// 生成 ECDSA 密钥对（P-256 曲线）
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("生成密钥失败: %v", err))
	}

	publicKey := privateKey.PublicKey

	// 将公钥转换为未压缩格式 (0x04 + 32字节X + 32字节Y)
	// elliptic.Marshal 返回未压缩格式的公钥
	publicKeyBytes := elliptic.Marshal(elliptic.P256(), publicKey.X, publicKey.Y)
	publicKeyBase64 := base64.RawURLEncoding.EncodeToString(publicKeyBytes)

	// 将私钥转换为 base64 URL 编码
	privateKeyBytes := privateKey.D.Bytes()
	privateKeyBase64 := base64.RawURLEncoding.EncodeToString(privateKeyBytes)

	fmt.Println("=== VAPID 密钥生成成功 ===")
	fmt.Println()
	fmt.Println("请将以下密钥添加到环境变量中：")
	fmt.Println()
	fmt.Printf("export VAPID_PUBLIC_KEY=%s\n", publicKeyBase64)
	fmt.Printf("export VAPID_PRIVATE_KEY=%s\n", privateKeyBase64)
	fmt.Println()
	fmt.Println("或者创建 .env 文件（如果使用环境变量加载工具）：")
	fmt.Println()
	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", publicKeyBase64)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", privateKeyBase64)
	fmt.Println()
}
