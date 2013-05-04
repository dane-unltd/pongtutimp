package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

type Model uint32

const (
	Paddle Model = 1
	Ball   Model = 2
)

const (
	Up   Action = 0
	Down Action = 1
)

type GameState struct {
	pos, vel, size []Vec
	model          []Model
	score          []uint32
	players        []PlayerId
}

func NewGameState() *GameState {
	st := &GameState{}
	st.pos = make([]Vec, 3)
	st.vel = make([]Vec, 3)
	st.size = make([]Vec, 3)
	st.model = make([]Model, 3)

	st.score = make([]uint32, 2)
	st.players = make([]PlayerId, 2)

	return st
}

var state = NewGameState()
var stateOld = NewGameState()

func init() {
	state.model[0] = Paddle
	state.pos[0] = Vec{-75, 0, 0}
	state.size[0] = Vec{5, 20, 10}

	state.model[1] = Paddle
	state.pos[1] = Vec{75, 0, 0}
	state.size[1] = Vec{5, 20, 10}

	state.model[2] = Ball
	state.size[2] = Vec{20, 20, 20}
}

func copyState() {
	copy(stateOld.pos, state.pos)
	copy(stateOld.vel, state.vel)
	copy(stateOld.size, state.size)
	copy(stateOld.model, state.model)

	copy(stateOld.score, state.score)
	copy(stateOld.players, state.players)
}

func updateSimulation() {
	processInput()
	collisionCheck()
	move()
}

func move() {
	for i := range state.pos {
		state.pos[i].Add(&state.pos[i], &state.vel[i])
	}
}

func processInput() {
	if state.players[0] == 0 || state.players[1] == 0 {
		return
	}

	newVel := 0.0
	if active(state.players[0], Up) {
		newVel += 5
	}
	if active(state.players[0], Down) {
		newVel -= 5
	}
	state.vel[0][1] = newVel

	newVel = 0.0
	if active(state.players[1], Up) {
		newVel += 5
	}
	if active(state.players[1], Down) {
		newVel -= 5
	}
	state.vel[1][1] = newVel
}

const FieldHeight = 120

func collisionCheck() {
	for i := range state.pos {
		if state.pos[i][1] > FieldHeight/2-state.size[i][1]/2 {
			state.pos[i][1] = FieldHeight/2 - state.size[i][1]/2
			if state.vel[i][1] > 0 {
				state.vel[i][1] = -state.vel[i][1]
			}
		}
		if state.pos[i][1] < -FieldHeight/2+state.size[i][1]/2 {
			state.pos[i][1] = -FieldHeight/2 + state.size[i][1]/2
			if state.vel[i][1] < 0 {
				state.vel[i][1] = -state.vel[i][1]
			}
		}
	}

	rSq := state.size[2][0] / 2
	rSq *= rSq
	for i := 0; i < 2; i++ {
		//v points from the center of the paddel to the point on the
		//border of the paddel which is closest to the sphere
		v := Vec{}
		v.Sub(&state.pos[2], &state.pos[i])
		v.Clamp(&state.size[i])

		//d is the vector between the closest points on the paddle and
		//the sphere
		d := Vec{}
		d.Sub(&state.pos[2], &state.pos[i])
		d.Sub(&d, &v)

		distSq := d.Nrm2Sq()
		if distSq < rSq {
			//move the sphere in direction of d to remove the
			//penetration
			dPos := Vec{}
			dPos.Scale(math.Sqrt(rSq/distSq)-1, &d)
			state.pos[2].Add(&state.pos[2], &dPos)

			dotPr := Dot(&state.vel[2], &d)
			if dotPr < 0 {
				d.Scale(-2*dotPr/distSq, &d)
				state.vel[2].Add(&state.vel[2], &d)
			}
		}
	}

	if state.pos[2][0] < -100 {
		state.pos[2] = Vec{0, 0, 0}
		state.vel[2] = Vec{2, 3, 0}
		state.score[1]++
	} else if state.pos[2][0] > 100 {
		state.pos[2] = Vec{0, 0, 0}
		state.vel[2] = Vec{-2, 3, 0}
		state.score[0]++
	}
}

func login(id PlayerId) {
	if state.players[0] == 0 {
		state.players[0] = id
		if state.players[1] != 0 {
			startGame()
		}
		return
	}
	if state.players[1] == 0 {
		state.players[1] = id
		startGame()
	}
}

func startGame() {
	state.score[0] = 0
	state.score[1] = 0
	state.vel[2] = Vec{2, 3, 0}
}

func disconnect(id PlayerId) {
	if state.players[0] == id {
		state.players[0] = 0
		stopGame()
	} else if state.players[1] == id {
		state.players[1] = 0
		stopGame()
	}
}

func stopGame() {
	state.pos[0] = Vec{-75, 0, 0}
	state.pos[1] = Vec{75, 0, 0}
	state.pos[2] = Vec{0, 0, 0}
	state.vel[2] = Vec{0, 0, 0}
}

func serializeVecSlice(buf io.Writer, serAll bool, sl, slOld []Vec) {
	bitMask := make([]byte, 1)
	bufTemp := &bytes.Buffer{}
	for i := range sl {
		if serAll || !sl[i].Equals(&slOld[i]) {
			bitMask[0] |= 1 << uint(i)
			binary.Write(bufTemp, binary.LittleEndian, &sl[i])
		}
	}
	buf.Write(bitMask)
	buf.Write(bufTemp.Bytes())
}

func serialize(buf io.Writer, serAll bool) {
	bitMask := make([]byte, 1)
	bufTemp := &bytes.Buffer{}
	for i := range state.model {
		if serAll || state.model[i] != stateOld.model[i] {
			bitMask[0] |= 1 << uint(i)
			binary.Write(bufTemp, binary.LittleEndian,
				state.model[i])
		}
	}
	buf.Write(bitMask)
	buf.Write(bufTemp.Bytes())

	serializeVecSlice(buf, serAll, state.pos, stateOld.pos)
	serializeVecSlice(buf, serAll, state.vel, stateOld.vel)
	serializeVecSlice(buf, serAll, state.size, stateOld.size)

	bitMask[0] = 0
	bufTemp.Reset()
	for i := range state.score {
		if serAll || state.score[i] != stateOld.score[i] {
			bitMask[0] |= 1 << uint(i)
			binary.Write(bufTemp, binary.LittleEndian,
				state.score[i])
		}
	}
	buf.Write(bitMask)
	buf.Write(bufTemp.Bytes())

	bitMask[0] = 0
	bufTemp.Reset()
	for i := range state.players {
		if serAll || state.players[i] != stateOld.players[i] {
			bitMask[0] |= 1 << uint(i)
			binary.Write(bufTemp, binary.LittleEndian,
				state.players[i])
		}
	}
	buf.Write(bitMask)
	buf.Write(bufTemp.Bytes())
}
