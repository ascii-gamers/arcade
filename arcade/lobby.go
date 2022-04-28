package arcade

import (
	"math/rand"
	"sync"

	"github.com/google/uuid"
)

type Lobby struct {
	sync.RWMutex

	ID        string
	Name      string
	Code      string
	Private   bool
	GameType  string
	Capacity  int
	NumFull   int
	PlayerIDs []string
	HostID    string
}

func NewLobby(name string, private bool, gameType string, capacity int, hostID string) *Lobby {
	lobby := &Lobby{
		ID:        uuid.NewString(),
		Name:      name,
		Private:   private,
		GameType:  gameType,
		Capacity:  capacity,
		NumFull:   1,
		PlayerIDs: []string{hostID},
		HostID:    hostID,
	}

	if private {
		lobby.Code = generateCode()
	}

	return lobby
}

func (l *Lobby) AddPlayer(playerID string) {
	l.PlayerIDs = append(l.PlayerIDs, playerID)
}

func generateCode() string {
	var code string

	for i := 0; i < 4; i++ {
		v := rand.Intn(25)
		code += string(letters[v])
	}

	return code
}
