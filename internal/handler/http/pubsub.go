package pubsub

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	entity "github.com/luqmanahmads/chatapp/internal/entity"
	"nhooyr.io/websocket"
)

type chatUsecase interface {
	SendChat(ctx context.Context, message entity.ChatMessage) error
	ReadChat(ctx context.Context, message entity.ChatMessage) (chan entity.ChatMessage, error)
}

type publishSubscribeHandler struct {
	mutex         sync.RWMutex
	subscriberMap map[string]subscriber
	logf          func(f string, v ...interface{})
	chatUsecase   chatUsecase
}

func New(chatUsecase chatUsecase) *publishSubscribeHandler {
	cs := &publishSubscribeHandler{
		subscriberMap: map[string]subscriber{},
		logf:          log.Printf,
		chatUsecase:   chatUsecase,
	}

	return cs
}

func (h *publishSubscribeHandler) HandleWelcome(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome chatapp!"))
}

func (h *publishSubscribeHandler) HandlePublish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logf("err read all: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var publishParam entity.PublishParam
	err = json.Unmarshal(body, &publishParam)
	if err != nil {
		h.logf("err unmarshall: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	h.logf("received: sender: %s receiver: %s message: %s", publishParam.Sender, publishParam.Receiver, publishParam.Message)
	subscriber, ok := h.getSubscriber(publishParam.Receiver)
	if !ok {
		h.logf("receiver not online")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	select {
	case subscriber.msgCh <- []byte(publishParam.Message):
	case <-ctx.Done():
		h.logf("send message to receiver timeout")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func (h *publishSubscribeHandler) HandleSubscribe(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	wsconn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.logf("failed websocket accept: %s", err.Error())
		wsconn.Close(websocket.StatusProtocolError, "cannot accept")
		return
	}
	defer wsconn.CloseNow()

	typ, msg, err := wsconn.Read(r.Context())
	if err != nil {
		h.logf("err wsconn read: %s", err.Error())
		wsconn.Close(websocket.StatusProtocolError, "cannot read ws message")
		return
	}

	if typ != websocket.MessageText {
		h.logf("invalid message type")
		wsconn.Close(websocket.StatusPolicyViolation, "invalid message type binary")
		return
	}

	var subscribeParam entity.SubscribeParam
	err = json.Unmarshal(msg, &subscribeParam)
	if err != nil {
		h.logf("err unmarshall: %s", err.Error())
		wsconn.Close(websocket.StatusUnsupportedData, "unmarshall failed")
		return
	}

	subscriber := h.addSubscriber(subscribeParam.Subscriber)
	defer h.deleteSubscriber(subscriber.name)

	h.logf("%s is online", subscriber.name)

	ctx := wsconn.CloseRead(r.Context())

	for {
		select {
		case msg := <-subscriber.msgCh:
			err = writeWithTimeout(ctx, wsconn, msg)
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *publishSubscribeHandler) HandlePublishV2(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logf("err read all: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var publishParam entity.PublishParam
	err = json.Unmarshal(body, &publishParam)
	if err != nil {
		h.logf("err unmarshall: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = h.chatUsecase.SendChat(r.Context(), entity.ChatMessage(publishParam))
	if err != nil {
		h.logf("err publish: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func (h *publishSubscribeHandler) HandleSubscribeV2(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	wsconn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.logf("failed websocket accept: %s", err.Error())
		wsconn.Close(websocket.StatusProtocolError, "cannot accept")
		return
	}
	defer wsconn.CloseNow()

	typ, msg, err := wsconn.Read(r.Context())
	if err != nil {
		h.logf("err wsconn read: %s", err.Error())
		wsconn.Close(websocket.StatusProtocolError, "cannot read ws message")
		return
	}

	if typ != websocket.MessageText {
		h.logf("invalid message type")
		wsconn.Close(websocket.StatusPolicyViolation, "invalid message type binary")
		return
	}

	var subscribeParam entity.SubscribeParam
	err = json.Unmarshal(msg, &subscribeParam)
	if err != nil {
		h.logf("err unmarshall: %s", err.Error())
		wsconn.Close(websocket.StatusUnsupportedData, "unmarshall failed")
		return
	}

	ctx := wsconn.CloseRead(r.Context())

	readChan, err := h.chatUsecase.ReadChat(ctx, entity.ChatMessage{Receiver: subscribeParam.Subscriber})
	if err != nil {
		h.logf("err read chat: %s", err.Error())
		wsconn.Close(websocket.StatusInternalError, "read chat failed")
		return
	}

	for {
		select {
		case msg := <-readChan:
			bytes, err := json.Marshal(msg)
			if err != nil {
				return
			}

			err = writeWithTimeout(ctx, wsconn, bytes)
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *publishSubscribeHandler) getSubscriber(name string) (subscriber, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	subscriber, ok := h.subscriberMap[name]
	return subscriber, ok
}

func (h *publishSubscribeHandler) addSubscriber(name string) subscriber {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	subscriber := subscriber{
		name:  name,
		msgCh: make(chan []byte),
	}

	h.subscriberMap[name] = subscriber
	return subscriber
}

func (h *publishSubscribeHandler) deleteSubscriber(name string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.subscriberMap, name)
}

func writeWithTimeout(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, msg)
}
