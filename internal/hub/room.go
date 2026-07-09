package hub

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

type RoomStatus string

const (
	RoomStatusActive RoomStatus = "active"
	RoomStatusEnded  RoomStatus = "ended"
)

type Room struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	HostUser  string     `json:"host_user"`
	JoinCode  string     `json:"join_code"`
	JoinToken string     `json:"join_token"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	MaxMembers int       `json:"max_members"`
	Status    RoomStatus `json:"status"`
	Channels  []string   `json:"channels"`
	Members   []RoomMember `json:"members"`
}

type RoomMember struct {
	Username  string    `json:"username"`
	UserID    string    `json:"user_id,omitempty"`
	JoinedAt  time.Time `json:"joined_at"`
	Connected bool      `json:"connected"`
}

type RoomOptions struct {
	Name       string
	TTL        time.Duration
	MaxMembers int
}

func DefaultRoomOptions() RoomOptions {
	return RoomOptions{
		Name:       "",
		TTL:        8 * time.Hour,
		MaxMembers: 20,
	}
}

func RoomGeneralChannel(roomID string) string {
	return fmt.Sprintf("room-%s-general", roomID)
}

func (h *Hub) GetRoom(roomID string) (*Room, bool) {
	if h == nil {
		return nil, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[roomID]
	return r, ok
}

func (h *Hub) GetRoomByJoinCode(code string) (*Room, bool) {
	if h == nil {
		return nil, false
	}
	c := normalizeRoomJoinCode(code)
	if c == "" {
		return nil, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	roomID, ok := h.roomsByCode[c]
	if !ok {
		return nil, false
	}
	r, ok := h.rooms[roomID]
	return r, ok
}

func (h *Hub) CreateRoom(hostUser string, opts RoomOptions) (*Room, error) {
	if h == nil {
		return nil, fmt.Errorf("nil hub")
	}
	host := strings.TrimSpace(hostUser)
	if host == "" {
		return nil, fmt.Errorf("host_user required")
	}
	if opts.MaxMembers <= 0 {
		opts.MaxMembers = DefaultRoomOptions().MaxMembers
	}
	if opts.TTL <= 0 {
		opts.TTL = DefaultRoomOptions().TTL
	}

	now := time.Now()
	roomID := uuid.New().String()

	h.mu.Lock()
	defer h.mu.Unlock()

	joinCode, err := h.generateUniqueRoomJoinCodeLocked()
	if err != nil {
		return nil, err
	}
	joinToken, err := generateRoomJoinToken()
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(opts.Name)

	room := &Room{
		ID:         roomID,
		Name:       name,
		HostUser:   host,
		JoinCode:   joinCode,
		JoinToken:  joinToken,
		CreatedAt:  now,
		ExpiresAt:  now.Add(opts.TTL),
		MaxMembers: opts.MaxMembers,
		Status:     RoomStatusActive,
		Channels:   []string{RoomGeneralChannel(roomID)},
		Members:    []RoomMember{},
	}
	h.rooms[roomID] = room
	h.roomsByCode[joinCode] = roomID
	return room, nil
}

func (h *Hub) JoinRoom(joinCode, username string) (*Room, error) {
	if h == nil {
		return nil, fmt.Errorf("nil hub")
	}
	code := normalizeRoomJoinCode(joinCode)
	user := slugUsername(username)
	if code == "" {
		return nil, fmt.Errorf("join_code required")
	}
	if user == "" {
		return nil, fmt.Errorf("username required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	roomID, ok := h.roomsByCode[code]
	if !ok {
		return nil, fmt.Errorf("invalid join code")
	}
	room, ok := h.rooms[roomID]
	if !ok || room == nil {
		return nil, fmt.Errorf("room unavailable")
	}
	if room.Status != RoomStatusActive {
		return nil, fmt.Errorf("room ended")
	}
	if time.Now().After(room.ExpiresAt) {
		room.Status = RoomStatusEnded
		delete(h.roomsByCode, room.JoinCode)
		return nil, fmt.Errorf("room expired")
	}
	if room.MaxMembers > 0 && len(room.Members) >= room.MaxMembers {
		return nil, fmt.Errorf("room full")
	}
	for i := range room.Members {
		if strings.EqualFold(slugUsername(room.Members[i].Username), user) {
			return room, nil
		}
	}
	room.Members = append(room.Members, RoomMember{
		Username:  username,
		JoinedAt:  time.Now(),
		Connected: false,
	})
	return room, nil
}

func (h *Hub) LeaveRoom(roomID, username string) {
	if h == nil {
		return
	}
	user := slugUsername(username)
	if roomID == "" || user == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[roomID]
	if !ok || room == nil {
		return
	}
	out := room.Members[:0]
	for _, m := range room.Members {
		if strings.EqualFold(slugUsername(m.Username), user) {
			continue
		}
		out = append(out, m)
	}
	room.Members = out
}

func (h *Hub) EndRoom(roomID, hostUser string) error {
	if h == nil {
		return fmt.Errorf("nil hub")
	}
	if roomID == "" {
		return fmt.Errorf("room_id required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[roomID]
	if !ok || room == nil {
		return fmt.Errorf("room not found")
	}
	if hostUser != "" && !strings.EqualFold(slugUsername(room.HostUser), slugUsername(hostUser)) {
		return fmt.Errorf("only host may end room")
	}
	room.Status = RoomStatusEnded
	delete(h.roomsByCode, room.JoinCode)
	return nil
}

func (h *Hub) ListActiveRooms(hostUser string) []*Room {
	if h == nil {
		return nil
	}
	host := slugUsername(hostUser)
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*Room, 0)
	now := time.Now()
	for _, r := range h.rooms {
		if r == nil || r.Status != RoomStatusActive {
			continue
		}
		if now.After(r.ExpiresAt) {
			continue
		}
		if host != "" && !strings.EqualFold(slugUsername(r.HostUser), host) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (h *Hub) SetRoomMemberConnected(roomID, username string, connected bool) bool {
	if h == nil {
		return false
	}
	roomID = strings.TrimSpace(roomID)
	user := slugUsername(username)
	if roomID == "" || user == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[roomID]
	if !ok || room == nil {
		return false
	}
	changed := false
	for i := range room.Members {
		if strings.EqualFold(slugUsername(room.Members[i].Username), user) {
			if room.Members[i].Connected != connected {
				room.Members[i].Connected = connected
				changed = true
			}
		}
	}
	if changed {
		h.syncRoomChannelMembersLocked(room)
	}
	return changed
}

func (h *Hub) SyncRoomChannelMembers(roomID string) {
	if h == nil || roomID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[roomID]
	if !ok || room == nil {
		return
	}
	h.syncRoomChannelMembersLocked(room)
}

func (h *Hub) syncRoomChannelMembersLocked(room *Room) {
	if h == nil || room == nil {
		return
	}
	members := make([]string, 0, len(room.Members))
	for _, m := range room.Members {
		if strings.TrimSpace(m.Username) != "" {
			members = append(members, m.Username)
		}
	}
	for _, chName := range room.Channels {
		ch, ok := h.channels[chName]
		if !ok || ch == nil {
			continue
		}
		ch.RoomID = room.ID
		ch.Type = protocol.ChannelTypeRoom
		ch.HumanMembers = members
	}
}

func (h *Hub) generateUniqueRoomJoinCodeLocked() (string, error) {
	// Must be called under h.mu.
	for i := 0; i < 20; i++ {
		code, err := generateRoomJoinCode()
		if err != nil {
			return "", err
		}
		if _, exists := h.roomsByCode[code]; !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to allocate join code")
}

func (h *Hub) runRoomExpiryLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		h.expireRoomsOnce()
	}
}

func (h *Hub) expireRoomsOnce() {
	if h == nil {
		return
	}
	now := time.Now()
	var expired []*Room

	h.mu.Lock()
	for id, r := range h.rooms {
		if r == nil || r.Status != RoomStatusActive {
			continue
		}
		if now.After(r.ExpiresAt) {
			r.Status = RoomStatusEnded
			delete(h.roomsByCode, r.JoinCode)
			delete(h.rooms, id)
			expired = append(expired, r)
		}
	}
	h.mu.Unlock()

	// Best-effort cleanup: delete channels + clear history outside the lock.
	for _, r := range expired {
		for _, chName := range r.Channels {
			_ = h.DeleteChannel(chName)
		}
	}
}

func normalizeRoomJoinCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	c = strings.ReplaceAll(c, "-", "")
	c = strings.ReplaceAll(c, " ", "")
	if len(c) != 6 {
		return ""
	}
	for _, r := range c {
		if (r >= 'A' && r <= 'Z') || (r >= '2' && r <= '7') {
			continue
		}
		return ""
	}
	return c
}

func generateRoomJoinCode() (string, error) {
	// 5 bytes -> 8 base32 chars. We'll take first 6 for a short join code.
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	code := enc[:6]
	return normalizeRoomJoinCode(code), nil
}

func generateRoomJoinToken() (string, error) {
	// 20 bytes -> 32 base32 chars (no padding). Not currently used for auth, but stored for future hardening.
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

