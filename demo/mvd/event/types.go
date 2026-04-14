package event

import "github.com/osm/quake/common/item"

type Type string

const (
	TypeFrag        Type = "frag"
	TypePlayerState Type = "player_state"
	TypeShot        Type = "shot"
	TypeKTX         Type = "ktx"
)

type KTXAction string

const (
	KTXActionTook           KTXAction = "took"
	KTXActionTimer          KTXAction = "timer"
	KTXActionDrop           KTXAction = "drop"
	KTXActionBackpackPickup KTXAction = "bp"
)

type Vec3 struct {
	X float32
	Y float32
	Z float32
}

func (v Vec3) Equal(other Vec3) bool {
	return v.X == other.X && v.Y == other.Y && v.Z == other.Z
}

type Frag struct {
	Time   float64
	Victim string
	Killer string
	Weapon item.Item
	Pos    Vec3
}

type PlayerState struct {
	Time   float64
	Edict  int
	Player string
	Team   string
	Pos    Vec3

	ViewAngles Vec3

	Frags  int
	Health int
	Armor  int

	Weapon item.Item

	HasSG   bool
	HasNG   bool
	HasSSG  bool
	HasSNG  bool
	HasGL   bool
	HasRL   bool
	HasLG   bool
	HasQuad bool
	HasRing bool
	HasPent bool
}

type Shot struct {
	Time       float64
	Edict      int
	Player     string
	Team       string
	Pos        Vec3
	ViewAngles Vec3
	Weapon     item.Item
}

type KTXEvent struct {
	Time        float64
	Entity      int
	Seconds     float64
	Action      KTXAction
	ItemBits    int
	PlayerEdict int
}

type Event struct {
	Time  float64
	Type  Type
	Frag  *Frag
	State *PlayerState
	Shot  *Shot
	KTX   *KTXEvent
}

type Result struct {
	Events     []Event
	maxClients int
}
