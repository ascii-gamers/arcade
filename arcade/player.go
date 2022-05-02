package arcade

const (
	GameDone = "GameDone"
	Waiting  = "Waiting"
	Looking  = "Looking"
	Playing  = "Playing"
)

type Player struct {
	ClientID string
	Username string
	Status   string
	Host     bool
}
