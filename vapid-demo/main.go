package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const (
	subscriptionsFile = "data/subscriptions.json"
)

var (
	vapidPublicKey  string
	vapidPrivateKey string
	subscriptions   = make(map[string]*webpush.Subscription)
	subsMutex       sync.RWMutex // 保护 subscriptions map 的并发访问
)

type SubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type PushMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url,omitempty"`
}

func main() {
	// 尝试从 .env 文件加载环境变量（如果文件存在）
	if err := godotenv.Load(); err != nil {
		// .env 文件不存在不是错误，只是记录日志
		log.Printf("未找到 .env 文件，将使用系统环境变量: %v", err)
	} else {
		log.Println("已从 .env 文件加载环境变量")
	}

	// 从环境变量读取 VAPID 密钥
	vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		log.Fatal("请设置 VAPID_PUBLIC_KEY 和 VAPID_PRIVATE_KEY 环境变量")
	}

	// 加载已保存的订阅
	if err := loadSubscriptions(); err != nil {
		log.Printf("加载订阅失败（将使用空订阅列表）: %v", err)
	} else {
		subsMutex.RLock()
		count := len(subscriptions)
		subsMutex.RUnlock()
		if count > 0 {
			log.Printf("已从 %s 加载 %d 个订阅", subscriptionsFile, count)
		}
	}

	// 创建路由
	r := mux.NewRouter()

	// 提供前端静态文件
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	r.HandleFunc("/", serveIndex)

	// API 路由
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/vapid-public-key", getVapidPublicKey).Methods("GET")
	api.HandleFunc("/subscribe", subscribe).Methods("POST")
	api.HandleFunc("/unsubscribe", unsubscribe).Methods("POST")
	api.HandleFunc("/push", push).Methods("POST")

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("服务器启动在端口 %s", port)
	log.Printf("访问 http://localhost:%s 查看前端页面", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// 提供前端页面
func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

// 获取 VAPID 公钥
func getVapidPublicKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"publicKey": vapidPublicKey,
	})
}

// 处理订阅请求
func subscribe(w http.ResponseWriter, r *http.Request) {
	var subReq SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&subReq); err != nil {
		http.Error(w, "无效的订阅数据", http.StatusBadRequest)
		return
	}

	// 创建 webpush.Subscription
	subscription := &webpush.Subscription{
		Endpoint: subReq.Endpoint,
		Keys: webpush.Keys{
			P256dh: subReq.Keys.P256dh,
			Auth:   subReq.Keys.Auth,
		},
	}

	// 使用 endpoint 作为唯一标识符
	subsMutex.Lock()
	subscriptions[subReq.Endpoint] = subscription
	subsMutex.Unlock()

	log.Printf("新订阅: %s", subReq.Endpoint)
	log.Printf("当前订阅数: %d", len(subscriptions))

	// 保存订阅到文件
	if err := saveSubscriptions(); err != nil {
		log.Printf("保存订阅失败: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "订阅成功",
	})
}

// 处理取消订阅请求
func unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		http.Error(w, "endpoint 不能为空", http.StatusBadRequest)
		return
	}

	// 从内存中删除订阅
	subsMutex.Lock()
	_, exists := subscriptions[req.Endpoint]
	if exists {
		delete(subscriptions, req.Endpoint)
		log.Printf("已取消订阅: %s", req.Endpoint)
	}
	subsMutex.Unlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "not_found",
			"message": "订阅不存在",
		})
		return
	}

	// 保存更新后的订阅到文件
	if err := saveSubscriptions(); err != nil {
		log.Printf("保存订阅失败: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "取消订阅成功",
	})
}

// 处理推送请求
func push(w http.ResponseWriter, r *http.Request) {
	var pushMsg PushMessage
	if err := json.NewDecoder(r.Body).Decode(&pushMsg); err != nil {
		http.Error(w, "无效的推送消息", http.StatusBadRequest)
		return
	}

	// 构建推送通知数据
	notification := map[string]interface{}{
		"title":     pushMsg.Title,
		"body":      pushMsg.Body,
		"icon":      pushMsg.Icon,
		"url":       pushMsg.URL,
		"timestamp": time.Now().Unix(),
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		http.Error(w, "无法序列化通知", http.StatusInternalServerError)
		return
	}

	// 向所有订阅者发送推送
	successCount := 0
	failCount := 0
	removedEndpoints := make([]string, 0)

	subsMutex.RLock()
	subsCopy := make(map[string]*webpush.Subscription)
	for k, v := range subscriptions {
		subsCopy[k] = v
	}
	subsMutex.RUnlock()

	for endpoint, subscription := range subsCopy {
		resp, err := webpush.SendNotification(notificationJSON, subscription, &webpush.Options{
			Subscriber:      "mailto:admin@example.com", // VAPID 要求
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			TTL:             30,
		})

		if err != nil {
			log.Printf("推送失败到 %s: %v", endpoint, err)
			failCount++
			// 如果订阅已失效，从列表中移除
			if resp != nil && resp.StatusCode == 410 {
				removedEndpoints = append(removedEndpoints, endpoint)
			}
			continue
		}

		resp.Body.Close()
		successCount++
		log.Printf("推送成功到 %s", endpoint)
	}

	// 移除失效的订阅
	if len(removedEndpoints) > 0 {
		subsMutex.Lock()
		for _, endpoint := range removedEndpoints {
			delete(subscriptions, endpoint)
			log.Printf("已移除失效订阅: %s", endpoint)
		}
		subsMutex.Unlock()

		// 保存更新后的订阅
		if err := saveSubscriptions(); err != nil {
			log.Printf("保存订阅失败: %v", err)
		}
	}

	subsMutex.RLock()
	totalSubs := len(subscriptions)
	subsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"success": successCount,
		"failed":  failCount,
		"total":   totalSubs,
		"message": fmt.Sprintf("已推送 %d 条消息，成功 %d，失败 %d", totalSubs, successCount, failCount),
	})
}

// 加载订阅从文件
func loadSubscriptions() error {
	// 检查文件是否存在
	if _, err := os.Stat(subscriptionsFile); os.IsNotExist(err) {
		log.Printf("订阅文件不存在，将创建新文件: %s", subscriptionsFile)
		return nil
	}

	// 读取文件
	data, err := os.ReadFile(subscriptionsFile)
	if err != nil {
		return fmt.Errorf("读取订阅文件失败: %w", err)
	}

	// 如果文件为空，返回空订阅
	if len(data) == 0 {
		return nil
	}

	// 解析 JSON
	var subsData map[string]struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(data, &subsData); err != nil {
		return fmt.Errorf("解析订阅文件失败: %w", err)
	}

	// 转换为 webpush.Subscription
	subsMutex.Lock()
	defer subsMutex.Unlock()
	subscriptions = make(map[string]*webpush.Subscription)
	for endpoint, subData := range subsData {
		subscriptions[endpoint] = &webpush.Subscription{
			Endpoint: subData.Endpoint,
			Keys: webpush.Keys{
				P256dh: subData.Keys.P256dh,
				Auth:   subData.Keys.Auth,
			},
		}
	}

	return nil
}

// 保存订阅到文件
func saveSubscriptions() error {
	// 确保目录存在
	dir := filepath.Dir(subscriptionsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 准备要保存的数据
	subsMutex.RLock()
	subsData := make(map[string]struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	})

	for endpoint, subscription := range subscriptions {
		subsData[endpoint] = struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				P256dh string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}{
			Endpoint: subscription.Endpoint,
			Keys: struct {
				P256dh string `json:"p256dh"`
				Auth   string `json:"auth"`
			}{
				P256dh: subscription.Keys.P256dh,
				Auth:   subscription.Keys.Auth,
			},
		}
	}
	subsMutex.RUnlock()

	// 序列化为 JSON
	data, err := json.MarshalIndent(subsData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化订阅数据失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(subscriptionsFile, data, 0644); err != nil {
		return fmt.Errorf("写入订阅文件失败: %w", err)
	}

	return nil
}
