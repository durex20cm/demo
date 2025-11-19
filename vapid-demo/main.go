package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gorilla/mux"
)

var (
	vapidPublicKey  string
	vapidPrivateKey string
	subscriptions   = make(map[string]*webpush.Subscription)
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
	// 从环境变量或配置文件读取 VAPID 密钥
	vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		log.Fatal("请设置 VAPID_PUBLIC_KEY 和 VAPID_PRIVATE_KEY 环境变量")
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
	subscriptions[subReq.Endpoint] = subscription

	log.Printf("新订阅: %s", subReq.Endpoint)
	log.Printf("当前订阅数: %d", len(subscriptions))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "订阅成功",
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

	for endpoint, subscription := range subscriptions {
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
				delete(subscriptions, endpoint)
			}
			continue
		}

		resp.Body.Close()
		successCount++
		log.Printf("推送成功到 %s", endpoint)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"success": successCount,
		"failed":  failCount,
		"total":   len(subscriptions),
		"message": fmt.Sprintf("已推送 %d 条消息，成功 %d，失败 %d", len(subscriptions), successCount, failCount),
	})
}
