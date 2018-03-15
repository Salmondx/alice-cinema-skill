package main

type Location struct {
	InProgress bool
	Completed  bool
	Subway     string
	City       string
}

type LocationStorage interface {
	Get(userID string) *Location
	Save(userID string, location *Location)
}

type InMemoryStorage struct {
	store map[string]*Location
}

func NewStorage() *InMemoryStorage {
	return &InMemoryStorage{make(map[string]*Location)}
}

func (s *InMemoryStorage) Get(userID string) *Location {
	if loc, ok := s.store[userID]; ok {
		return loc
	}
	return &Location{}
}

func (s *InMemoryStorage) Save(userID string, location *Location) {
	s.store[userID] = location
}
