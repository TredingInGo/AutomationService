package users

import (
	"context"
	smartapi "github.com/TredingInGo/smartapi"
	"sync"
)

type ActiveUsers struct {
	userInfo map[string]*UserInfo
	mu       sync.Mutex
}

type UserInfo struct {
	ApiClient         *smartapi.Client
	Session           smartapi.UserSession
	CancelFunc        context.CancelFunc
	Ctx               context.Context
	IsIntraDayRunning bool
}

func New() ActiveUsers {
	return ActiveUsers{
		userInfo: make(map[string]*UserInfo),
		mu:       sync.Mutex{},
	}
}

func (au *ActiveUsers) Get(clientID string) (*UserInfo, bool) {
	au.mu.Lock()
	defer au.mu.Unlock()

	user, exist := au.userInfo[clientID]

	return user, exist
}

func (au *ActiveUsers) Set(clientID string, user *UserInfo) bool {
	_, exists := au.Get(clientID)
	if exists {
		return false
	}

	au.mu.Lock()
	defer au.mu.Unlock()

	au.userInfo[clientID] = user

	return true
}

func (au *ActiveUsers) Remove(clientID string) {
	au.mu.Lock()
	defer au.mu.Unlock()

	_, exist := au.userInfo[clientID]
	if exist {
		delete(au.userInfo, clientID)
	}
}

func (au *ActiveUsers) Update(clientID string, user *UserInfo) {
	au.mu.Lock()
	defer au.mu.Unlock()

	au.userInfo[clientID] = user
}
