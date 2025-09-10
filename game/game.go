package game

import (
	"math/rand"
	"sync"
	"time"
)

const (
	DIR_NONE  = 0
	DIR_UP    = 1
	DIR_RIGHT = 2
	DIR_DOWN  = 3
	DIR_LEFT  = 4
)

type TailSegment struct {
	X int
	Y int
}

type Player struct {
	Active       bool
	X            int
	Y            int
	Dir          int
	Tail         []*TailSegment
	Speed        int
	LastMoveTime int64
}

type Fruit struct {
	Active    bool
	X         int
	Y         int
	SpawnTime int64
}

type Game struct {
	Rows          int
	Cols          int
	Player        Player
	Fruit         Fruit
	Score         int
	LastFrameTime int64
	ShouldRedraw  bool
	StartTime     int64
	IsPaused      bool
	ShowHelp      bool
	Lock          sync.Mutex
}

func NewGame(rows, cols int) *Game {
	return &Game{
		Rows: rows,
		Cols: cols,
		Player: Player{
			Active: true,
			X:      0,
			Y:      0,
			Dir:    DIR_NONE,
			Tail:   []*TailSegment{},
			Speed:  5,
		},
		Fruit: Fruit{
			Active: false,
			X:      0,
			Y:      0,
		},
		LastFrameTime: time.Now().UnixMilli(),
		ShouldRedraw:  true,
		StartTime:     time.Now().UnixMilli(),
		IsPaused:      false,
		ShowHelp:      true,
		Lock:          sync.Mutex{},
	}
}

func (state *Game) Update(renderer func(*Game) error, forceRedraw bool) {
	state.Lock.Lock()
	defer state.Lock.Unlock()

	if !state.IsPaused {
		timeSinceLastPlayerMove := time.Now().UnixMilli() - state.Player.LastMoveTime

		if state.Player.Dir != DIR_NONE && (state.Player.LastMoveTime == 0 || timeSinceLastPlayerMove > int64(1000/state.Player.Speed)) {
			playerX := state.Player.X
			playerY := state.Player.Y

			switch state.Player.Dir {
			case DIR_UP:
				state.Player.Y--
			case DIR_RIGHT:
				state.Player.X++
			case DIR_DOWN:
				state.Player.Y++
			case DIR_LEFT:
				state.Player.X--
			}

			if state.checkPlayerHitsBounds() {
				state.reset()
				return
			}

			if state.checkPlayerHitsTail() {
				state.reset()
				return
			}

			if len(state.Player.Tail) > 0 {
				x := playerX
				y := playerY

				for _, tail := range state.Player.Tail {
					tempX := tail.X
					tempY := tail.Y
					tail.X = x
					tail.Y = y
					x = tempX
					y = tempY
				}
			}

			state.Player.LastMoveTime = time.Now().UnixMilli()
			state.ShouldRedraw = true
		}

		if state.Fruit.Active {
			if state.checkPlayerHitsFruit() {
				state.Score++
				state.Fruit.Active = false
				state.spawnFruit()
				state.growPlayerTail()
			}
		} else {
			if time.Now().UnixMilli()-state.Fruit.SpawnTime > 3000 {
				state.spawnFruit()
			}
		}
	}

	if state.ShouldRedraw || forceRedraw {
		err := renderer(state)

		if err != nil {
			return
		}

		state.ShouldRedraw = false
		state.LastFrameTime = time.Now().UnixMilli()
	}
}

func (state *Game) reset() {
	state.Player.X = 0
	state.Player.Y = 0
	state.Player.Dir = DIR_NONE
	state.Player.Tail = []*TailSegment{}
	state.Fruit.Active = false
	state.Score = 0
	state.StartTime = time.Now().UnixMilli()
	state.ShouldRedraw = true
}

func (state *Game) checkPlayerHitsBounds() bool {
	return state.Player.X < 0 || state.Player.X >= state.Cols || state.Player.Y < 0 || state.Player.Y >= state.Rows
}

func (state *Game) checkPlayerHitsFruit() bool {
	return state.Player.X == state.Fruit.X && state.Player.Y == state.Fruit.Y
}

func (state *Game) checkPlayerHitsTail() bool {
	for _, tail := range state.Player.Tail {
		if state.Player.X == tail.X && state.Player.Y == tail.Y {
			return true
		}
	}

	return false
}

func (state *Game) growPlayerTail() {
	state.Player.Tail = append(state.Player.Tail, &TailSegment{
		X: state.Player.X,
		Y: state.Player.Y,
	})
	state.ShouldRedraw = true
}

func (state *Game) spawnFruit() {
	x, y := 0, 0
	attempts := 0

	for attempts < 1000 {
		attempts++

		x = 1 + rand.Int()%state.Cols - 1
		y = 1 + rand.Int()%state.Rows - 1

		if x != state.Player.X && y != state.Player.Y {
			state.Fruit.X = x
			state.Fruit.Y = y
			state.Fruit.SpawnTime = time.Now().UnixMilli()
			state.Fruit.Active = true
			state.ShouldRedraw = true
			break
		}
	}
}

func (state *Game) SetPlayerDir(dir int) {
	state.Lock.Lock()
	defer state.Lock.Unlock()
	state.Player.Dir = dir
}

func (state *Game) TogglePause() {
	state.Lock.Lock()
	defer state.Lock.Unlock()
	state.IsPaused = !state.IsPaused
	state.ShouldRedraw = true
}

func (state *Game) ToggleHelp() {
	state.Lock.Lock()
	defer state.Lock.Unlock()
	state.ShowHelp = !state.ShowHelp
	state.ShouldRedraw = true
}

func (state *Game) Restart() {
	state.Lock.Lock()
	defer state.Lock.Unlock()
	state.reset()
}
