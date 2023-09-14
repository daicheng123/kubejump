package exchange

import (
	"encoding/json"
	"k8s.io/klog/v2"
	"time"
)

func proxyRoom(room *Room, ch *redisChannel, userInputCh chan *RoomMessage) {
	maxIdleTime := time.Minute * 30
	tick := time.NewTicker(time.Second * 30)
	defer tick.Stop()
	defer func() {
		ch.manager.removeProxyRoomChan <- room
		err := ch.Close() // 关闭连接
		if err != nil {
			klog.Errorf("Redis channel close err: %s", err)
		}
		klog.Infof("Proxy redis room %s done", room.Id)
	}()
	active := time.Now()
	for {
		select {
		case <-room.Done():
			klog.Infof("Redis room %s done", ch.roomId)
			return

		case tickNow := <-tick.C:
			if !tickNow.After(active.Add(maxIdleTime)) {
				continue
			}
			klog.Errorf("Redis room %s exceed max idle time", ch.roomId)
			return
		case msg, ok := <-userInputCh:
			if !ok {
				return
			}
			if err := ch.sendMessage(msg); err != nil {
				klog.Errorf("Redis room %s send message err: %s", ch.roomId, err)
			}

		case redisMsg, ok := <-ch.subMsgCh:
			if !ok {
				klog.Infof("Redis room %s stop receive message", ch.roomId)
				return
			}
			var msg RoomMessage
			if err := json.Unmarshal(redisMsg.Message, &msg); err != nil {
				klog.Errorf("Redis proxy room %s message unmarshal err: %s", ch.roomId, err)
				continue
			}
			room.Broadcast(&msg)
		}
		active = time.Now()
	}
}

// 接受其他 koko 的数据 给 Room
func proxyUserCon(room *Room, ch *redisChannel) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	currentNumber := 1
	con := WrapperUserCon(ch)
	room.Subscribe(con)
	defer func() {
		room.UnSubscribe(con)
		err := ch.Close()
		if err != nil {
			klog.Errorf("Redis channel close err: %s", err)
		}
		klog.Infof("Proxy redis userCon for room %s done", room.Id)
	}()
	for {
		select {
		case <-tick.C:
			if currentNumber > 0 {
				continue
			}
			klog.Infof("Redis proxy userCon for room %s has no subscribers and exit", ch.roomId)
			return
		case number := <-ch.count:
			currentNumber += number

		case redisMsg, ok := <-ch.subMsgCh:
			if !ok {
				klog.Infof("Redis proxy userCon for room %s stop receive message", ch.roomId)
				return
			}
			var msg RoomMessage
			_ = json.Unmarshal(redisMsg.Message, &msg)
			room.Receive(&msg)
		}
	}
}
