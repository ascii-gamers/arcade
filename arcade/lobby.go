package arcade

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"arcade/labrpc"

	"github.com/google/uuid"
)

type Lobby struct {
	mu sync.RWMutex

	ID               string
	Name             string
	Code             string
	Private          bool
	GameType         string
	Capacity         int
	PlayerIDs        []string
	HostID           string
	Ping             int
	PlayerClientEnds labrpc.ClientEnd
}

func NewLobby(name string, private bool, gameType string, capacity int, hostID string) *Lobby {
	lobby := &Lobby{
		ID:        uuid.NewString(),
		Name:      name,
		Private:   private,
		GameType:  gameType,
		Capacity:  capacity,
		PlayerIDs: []string{hostID},
		HostID:    hostID,
	}

	if private {
		lobby.Code = generateCode()
	}

	return lobby
}

func (l *Lobby) AddPlayer(playerID string) {
	l.mu.Lock()
	l.PlayerIDs = append(l.PlayerIDs, playerID)
	l.mu.Unlock()
}

func (l *Lobby) RemovePlayer(playerID string) {
	l.mu.Lock()
	for i, v := range l.PlayerIDs {
		if v == playerID {
			l.PlayerIDs = append(l.PlayerIDs[:i], l.PlayerIDs[i+1:]...)
			break
		}
	}
	l.mu.Unlock()
}

func generateCode() string {
	var code string
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 4; i++ {
		v := rand.Intn(25)
		code += string(letters[v])
	}

	return code
}

func (l *Lobby) MarshalBinary() ([]byte, error) {
	return json.Marshal(l)
}
