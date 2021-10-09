package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type usersCreds map[string]string
type UsersStore struct {
	creds usersCreds
	mu    sync.Mutex
}

func NewUsersStore() *UsersStore {
	usersStore := &UsersStore{
		creds: make(usersCreds),
	}
	return usersStore
}

func (s *UsersStore) Set(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.creds[username] = password
}

func (s *UsersStore) Get(username string) (password string, ok bool) {
	password, ok = s.creds[username]
	return
}

func (s *UsersStore) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&s.creds); err != nil {
		return err
	}
	return nil
}

func (s *UsersStore) Save(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(s.creds); err != nil {
		return err
	}
	return nil
}
