package security

import (
	"fmt"
	"math/rand"
	"regexp"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type SecurityPolicy struct {
	mu                sync.RWMutex
	allowlist         []string
	blocklist         []string
	sandbox           bool
	approveBlocklist  bool
	maxFileSize       int64
	allowedExtensions []string
	blockedExtensions []string
}

type AutonomyLevel string

const (
	AutonomyLevelFull     AutonomyLevel = "full"
	AutonomyLevelApprove  AutonomyLevel = "approve"
	AutonomyLevelReadOnly AutonomyLevel = "read_only"
	AutonomyLevelDisabled AutonomyLevel = "disabled"
)

func NewSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		allowlist:         []string{},
		blocklist:         []string{},
		sandbox:           true,
		approveBlocklist:  false,
		maxFileSize:       100 * 1024 * 1024, // 100MB
		allowedExtensions: []string{},
		blockedExtensions: []string{},
	}
}

func (p *SecurityPolicy) SetAllowlist(patterns []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.allowlist = patterns
}

func (p *SecurityPolicy) SetBlocklist(patterns []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocklist = patterns
}

func (p *SecurityPolicy) SetSandbox(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sandbox = enabled
}

func (p *SecurityPolicy) IsAllowed(path string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.allowlist) > 0 {
		for _, pattern := range p.allowlist {
			if matchesPattern(path, pattern) {
				return true
			}
		}
		return false
	}

	if len(p.blocklist) > 0 {
		for _, pattern := range p.blocklist {
			if matchesPattern(path, pattern) {
				return false
			}
		}
	}

	return true
}

func (p *SecurityPolicy) IsSandboxEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sandbox
}

func (p *SecurityPolicy) CheckFileSize(size int64) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if size > p.maxFileSize {
		return fmt.Errorf("file size %d exceeds maximum %d", size, p.maxFileSize)
	}
	return nil
}

func (p *SecurityPolicy) CheckExtension(filename string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ext := getExtension(filename)

	if len(p.blockedExtensions) > 0 {
		for _, blocked := range p.blockedExtensions {
			if ext == blocked {
				return fmt.Errorf("extension %s is blocked", ext)
			}
		}
	}

	if len(p.allowedExtensions) > 0 {
		allowedExt := false
		for _, allowed := range p.allowedExtensions {
			if ext == allowed {
				allowedExt = true
				break
			}
		}
		if !allowedExt {
			return fmt.Errorf("extension %s is not allowed", ext)
		}
	}

	return nil
}

func matchesPattern(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = regexp.MustCompile(`\*`).ReplaceAllString(regexPattern, ".*")
	regexPattern = regexp.MustCompile(`\?`).ReplaceAllString(regexPattern, ".")

	matched, _ := regexp.MatchString("^"+regexPattern+"$", text)
	return matched
}

func getExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}
	return ""
}

type SecretStore struct {
	mu      sync.RWMutex
	secrets map[string]string
}

func NewSecretStore() *SecretStore {
	return &SecretStore{
		secrets: make(map[string]string),
	}
}

func (s *SecretStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[key] = value
}

func (s *SecretStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.secrets[key]
	return value, exists
}

func (s *SecretStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, key)
}

func (s *SecretStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.secrets))
	for key := range s.secrets {
		keys = append(keys, key)
	}

	return keys
}

func (s *SecretStore) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.secrets[key]
	return exists
}

type PairingGuard struct {
	mu           sync.RWMutex
	allowedUsers []string
	code         string
	enabled      bool
	usedCodes    map[string]bool
}

func NewPairingGuard(enabled bool, allowedUsers []string) *PairingGuard {
	if !enabled {
		return nil
	}

	return &PairingGuard{
		allowedUsers: allowedUsers,
		code:         generatePairingCode(),
		enabled:      enabled,
		usedCodes:    make(map[string]bool),
	}
}

func (g *PairingGuard) PairingCode() string {
	if g == nil {
		return ""
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.code
}

func (g *PairingGuard) VerifyCode(code string) bool {
	if g == nil {
		return false
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.code == code && !g.usedCodes[code] {
		g.usedCodes[code] = true
		return true
	}

	return false
}

func (g *PairingGuard) IsEnabled() bool {
	if g == nil {
		return false
	}

	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enabled
}

func (g *PairingGuard) AddUser(userID string) {
	if g == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	for _, u := range g.allowedUsers {
		if u == userID {
			return
		}
	}

	g.allowedUsers = append(g.allowedUsers, userID)
}

func (g *PairingGuard) RemoveUser(userID string) {
	if g == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	for i, u := range g.allowedUsers {
		if u == userID {
			g.allowedUsers = append(g.allowedUsers[:i], g.allowedUsers[i+1:]...)
			return
		}
	}
}

func (g *PairingGuard) IsAllowedUser(userID string) bool {
	if g == nil {
		return true
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.allowedUsers) == 0 {
		return false
	}

	for _, u := range g.allowedUsers {
		if u == "*" || u == userID {
			return true
		}
	}

	return false
}

func generatePairingCode() string {
	code := rand.Intn(900000) + 100000
	return fmt.Sprintf("%06d", code)
}
