// Service Worker 用于接收推送通知

// 安装 Service Worker
self.addEventListener('install', event => {
    console.log('Service Worker 安装中...');
    self.skipWaiting(); // 立即激活新的 Service Worker
});

// 激活 Service Worker
self.addEventListener('activate', event => {
    console.log('Service Worker 已激活');
    event.waitUntil(self.clients.claim()); // 立即控制所有客户端
});

// 监听推送事件
self.addEventListener('push', event => {
    console.log('收到推送消息', event);

    let notificationData = {
        title: '新通知',
        body: '您有一条新消息',
        icon: '/static/icon.png',
        badge: '/static/icon.png',
        tag: 'vapid-demo',
        requireInteraction: false
    };

    // 解析推送数据
    if (event.data) {
        try {
            const data = event.data.json();
            notificationData = {
                title: data.title || notificationData.title,
                body: data.body || notificationData.body,
                icon: data.icon || notificationData.icon,
                badge: data.icon || notificationData.badge,
                tag: data.tag || notificationData.tag,
                data: {
                    url: data.url || self.location.origin
                },
                requireInteraction: data.requireInteraction || false
            };
        } catch (e) {
            // 如果不是 JSON，尝试作为文本
            notificationData.body = event.data.text();
        }
    }

    // 显示通知
    event.waitUntil(
        self.registration.showNotification(notificationData.title, {
            body: notificationData.body,
            icon: notificationData.icon,
            badge: notificationData.badge,
            tag: notificationData.tag,
            data: notificationData.data,
            requireInteraction: notificationData.requireInteraction,
            vibrate: [200, 100, 200],
            actions: [
                {
                    action: 'open',
                    title: '打开'
                },
                {
                    action: 'close',
                    title: '关闭'
                }
            ]
        })
    );
});

// 处理通知点击事件
self.addEventListener('notificationclick', event => {
    console.log('通知被点击', event);

    event.notification.close();

    if (event.action === 'close') {
        return;
    }

    // 打开或聚焦到应用
    event.waitUntil(
        clients.matchAll({
            type: 'window',
            includeUncontrolled: true
        }).then(clientList => {
            // 如果已经有打开的窗口，聚焦它
            for (let i = 0; i < clientList.length; i++) {
                const client = clientList[i];
                if (client.url === event.notification.data.url && 'focus' in client) {
                    return client.focus();
                }
            }
            // 否则打开新窗口
            if (clients.openWindow) {
                return clients.openWindow(event.notification.data.url || '/');
            }
        })
    );
});

// 处理通知关闭事件
self.addEventListener('notificationclose', event => {
    console.log('通知已关闭', event);
});

