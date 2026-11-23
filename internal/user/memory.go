package user

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	pb "lastmile/gen/go/user"
)

// DemoPassword is the password for the seeded demo accounts.
const DemoPassword = "lastmile"

// DemoAccounts are created at startup whenever the service runs without
// Supabase, so a classroom demo works with no internet and no setup.
var DemoAccounts = []struct {
	ID, Email, Name string
	Role            pb.UserRole
}{
	{"demo-driver-asha", "driver@lastmile.demo", "Asha (driver)", pb.UserRole_USER_ROLE_DRIVER},
	{"demo-driver-ravi", "driver2@lastmile.demo", "Ravi (driver)", pb.UserRole_USER_ROLE_DRIVER},
	{"demo-rider-priya", "rider@lastmile.demo", "Priya (rider)", pb.UserRole_USER_ROLE_RIDER},
	{"demo-rider-kabir", "rider2@lastmile.demo", "Kabir (rider)", pb.UserRole_USER_ROLE_RIDER},
}

type memUser struct {
	user *pb.User
	hash []byte
}

// memoryStore is the in-memory auth backend used when Supabase is not
// configured. Passwords are bcrypt-hashed even here: demo code gets copied.
type memoryStore struct {
	mu      sync.RWMutex
	byEmail map[string]*memUser
	byID    map[string]*memUser
}

func newMemoryStore(seed bool) *memoryStore {
	m := &memoryStore{byEmail: map[string]*memUser{}, byID: map[string]*memUser{}}
	if seed {
		hash, _ := bcrypt.GenerateFromPassword([]byte(DemoPassword), bcrypt.MinCost)
		for _, a := range DemoAccounts {
			u := &memUser{user: &pb.User{Id: a.ID, Email: a.Email, Name: a.Name, Role: a.Role}, hash: hash}
			m.byEmail[a.Email] = u
			m.byID[a.ID] = u
		}
	}
	return m
}

func (m *memoryStore) signUp(email, password, name string, role pb.UserRole) (*pb.User, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, err
	}
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if role == pb.UserRole_USER_ROLE_UNSPECIFIED {
		role = pb.UserRole_USER_ROLE_RIDER
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[email]; exists {
		return nil, true, nil
	}
	u := &memUser{user: &pb.User{Id: uuid.New().String(), Email: email, Name: name, Role: role}, hash: hash}
	m.byEmail[email] = u
	m.byID[u.user.Id] = u
	return u.user, false, nil
}

func (m *memoryStore) signIn(email, password string) (*pb.User, bool) {
	m.mu.RLock()
	u, ok := m.byEmail[strings.ToLower(strings.TrimSpace(email))]
	m.mu.RUnlock()
	if !ok || bcrypt.CompareHashAndPassword(u.hash, []byte(password)) != nil {
		return nil, false
	}
	return u.user, true
}

func (m *memoryStore) get(id string) (*pb.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, false
	}
	return u.user, true
}

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "demo." + hex.EncodeToString(b)
}
