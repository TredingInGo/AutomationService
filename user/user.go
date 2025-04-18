package user

import (
	"context"
	smartapi "github.com/TredingInGo/smartapi"
	"log"
	"sync"
	"time"
)

type activeUsers struct {
	userInfo map[string]*UserInfo
	mu       sync.Mutex
}

type UserInfo struct {
	ApiClient            *smartapi.Client
	Session              smartapi.UserSession
	CancelFunc           context.CancelFunc
	Ctx                  context.Context
	IsIntraDayRunning    bool
	AliceBlueClientID    string
	AliceBlueSessionCode string
	UserId               string
	Name                 string
	AliceBlueBearerToken string
}

type Users interface {
	Get(clientID string) (*UserInfo, bool)
	Set(clientID string, user *UserInfo) bool
	Remove(clientID string)
	Update(clientID string, user *UserInfo)
	RemoveAll()
}

func New() Users {
	return &activeUsers{
		userInfo: make(map[string]*UserInfo),
		mu:       sync.Mutex{},
	}
}

func (au *activeUsers) Get(clientID string) (*UserInfo, bool) {
	au.mu.Lock()
	defer au.mu.Unlock()

	user, exist := au.userInfo[clientID]

	return user, exist
}

func (au *activeUsers) Set(clientID string, user *UserInfo) bool {

	au.mu.Lock()
	defer au.mu.Unlock()

	au.userInfo[clientID] = user

	return true
}

func (au *activeUsers) Remove(clientID string) {
	au.mu.Lock()
	defer au.mu.Unlock()

	_, exist := au.userInfo[clientID]
	if exist {
		delete(au.userInfo, clientID)
	}
}

func (au *activeUsers) Update(clientID string, user *UserInfo) {
	au.mu.Lock()
	defer au.mu.Unlock()

	au.userInfo[clientID] = user
}

func (au *activeUsers) RemoveAll() {
	au.mu.Lock()
	defer au.mu.Unlock()

	log.Println("Removing all user at : ", time.Now().Format("2006-01-02 15:04:05"))
	log.Println("Total user to remove: ", len(au.userInfo))

	for k, v := range au.userInfo {
		log.Println("removing: ", k)

		// cancelling the context, so that intra-day will be stopped
		v.CancelFunc()
		delete(au.userInfo, k)
	}
}
