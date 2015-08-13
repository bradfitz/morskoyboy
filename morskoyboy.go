package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Boat struct {
	name string
	len  uint8
}

var boats = []Boat{
	{"battleship", 4},
	{"destroyer1", 3},
	{"destroyer2", 3},
	{"cruiser1", 2},
	{"cruiser2", 2},
	{"cruiser3", 2},
	{"sailboat1", 1},
	{"sailboat2", 1},
	{"sailboat3", 1},
	{"sailboat4", 1},
}

type Cell uint8

const (
	MaskBoat  = 1 << 0
	MaskFired = 1 << 1
)

func (c Cell) IsBoat() bool  { return c&MaskBoat != 0 }
func (c Cell) IsFired() bool { return c&MaskFired != 0 }

func (b *Board) safeCell(x, y int) Cell {
	if x < 0 || y < 0 || x >= boardWidth || y >= boardHeight {
		return 0
	}
	return b[y][x]
}

func (b *Board) CellIcon(x, y int) byte {
	c := b[y][x]
	if c.IsFired() {
		if c.IsBoat() {
			return 'X'
		} else {
			return '.'
		}
	}
	for _, v := range []struct{ xo, yo int }{{-1, -1}, {1, 1}, {-1, 1}, {1, -1}} {
		if corner := b.safeCell(x+v.xo, y+v.yo); corner.IsFired() && corner.IsBoat() {
			return '~'
		}
	}
	return ' '
}

const (
	boardWidth  = 10
	boardHeight = 10
)

type Board [boardHeight][boardWidth]Cell // y -> x

func (b *Board) Fire(x, y int) {
	b[y][x] |= MaskFired
}

func (b *Board) IsBoat(x, y int) bool {
	return b[y][x].IsBoat()
}

func (b *Board) PlaceBoatCell(x, y int) {
	b[y][x] |= MaskBoat
}

func (b *Board) UnhitBoatCellsRemain() int {
	var n int
	for y := range b {
		for _, c := range b[y] {
			if c.IsBoat() && !c.IsFired() {
				n++
			}
		}
	}
	return n
}

func (b *Board) AllHit() bool { return b.UnhitBoatCellsRemain() == 0 }

// IsWater reports whether cell (x, y) is either water or is off the board.
func (b *Board) IsWater(x, y int) bool {
	// Anything off the edge of the world is water.
	if x < 0 || x >= boardWidth || y < 0 || y >= boardHeight {
		return true
	}
	return !b[y][x].IsBoat()
}

func (b *Board) PlaceBoat(x0, y0, x1, y1 int) bool {
	// Is there an existing boat there?
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			if x < 0 || x >= boardWidth || y < 0 || y >= boardHeight || b[y][x].IsBoat() {
				return false
			}
		}
	}

	// Is there water (or border) surrounding the boat?
	for x := x0 - 1; x <= x1+1; x++ {
		for y := y0 - 1; y <= y1+1; y++ {
			if !b.IsWater(x, y) {
				return false
			}
		}
	}

	// Place it.
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			b.PlaceBoatCell(x, y)
		}
	}
	return true
}

// +-+-+-+-+-+-+-+-+-+-+
// | | | | | | | | | | |
// +-+-+-+-+-+-+-+-+-+-+

type Screen [25][80]byte

func (s *Screen) Clear() {
	for y := range s {
		for x := range s[y] {
			s[y][x] = ' '
		}
	}
}

func (s *Screen) Print() {
	os.Stdout.Write(clear)
	for y := range s {
		fmt.Printf("%s\n", s[y][:])
	}
}

func (s *Screen) RenderBoard(b *Board, screenXoff, screenYoff int, all bool) {
	var lastY int
	for y := range b {
		sy := screenYoff + y*2 + 1
		s[sy][screenXoff] = '0' + byte(y)
		for x, c := range b[y] {
			sx := screenXoff + (x+1)*3
			s[screenYoff][sx] = 'A' + byte(x)
			tile := b.CellIcon(x, y)
			if all && tile == ' ' && c.IsBoat() {
				tile = 'B'
			}
			s[sy][sx] = tile
			s[sy][sx+1] = '|'
			s[sy+1][sx] = '-'
			s[sy+1][sx-1] = '-'
			s[sy+1][sx+1] = '+'
		}
		lastY = sy + 1
	}
	copy(s[lastY+2][screenXoff:], []byte(fmt.Sprintf("Boat parts remain: %d", b.UnhitBoatCellsRemain())))
}

var clear []byte

var (
	devMode = flag.Bool("dev", false, "dev mode; random boat placement and boats always visible")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	clear, _ = exec.Command("clear").Output()
	var s Screen

	var p1, p2 Board

	// placements
	var turn int // 0 == kate, 1 == brad
	players := [...]string{"Brad", "Kate"}
	boards := [...]*Board{&p1, &p2}
	for i, player := range players {
		for _, boat := range boats {
		Boat:
			b := boards[1-i]
			s.Clear()
			s.RenderBoard(b, 2, 2, true)
			s.Print()
			fmt.Printf("%s, %s (%d)> ", player, boat.name, boat.len)
			var in string
			if *devMode {
				in = randomPlacement()
			} else {
				if _, err := fmt.Scanf("%s\n", &in); err != nil {
					goto Boat
				}
			}
			in = strings.TrimSpace(strings.ToUpper(in))
			if boat.len == 1 && len(in) == 2 {
				in += "R"
			}
			if len(in) != 3 || in[0] < 'A' || in[0] > 'J' || in[1] < '0' || in[1] > '9' ||
				(in[2] != 'R' && in[2] != 'D') {
				if *devMode {
					log.Fatalf("bad input %q", in)
				} else {
					fmt.Printf("BAD INPUT %q\n", in)
					time.Sleep(1 * time.Second)
				}
				goto Boat
			}
			dir := in[2]
			var fx, fy int
			if dir == 'R' {
				fx = 1
			}
			if dir == 'D' {
				fy = 1
			}
			x := int(in[0] - 'A')
			y := int(in[1] - '0')
			if !b.PlaceBoat(x, y, x+fx*int(boat.len-1), y+fy*int(boat.len-1)) {
				if !*devMode {
					fmt.Println("CONFLICT")
					time.Sleep(1 * time.Second)
				}
				goto Boat
			}
		}
	}

Game:
	for {
		s.Clear()
		s.RenderBoard(&p1, 0, 0, *devMode)
		if !*devMode {
			s.RenderBoard(&p2, 40, 0, false)
		}
		s.Print()

		fmt.Printf("%s> ", players[turn])
		var in string
		if _, err := fmt.Scanf("%s\n", &in); err != nil {
			continue
		}
		in = strings.TrimSpace(strings.ToUpper(in))
		if len(in) != 2 || in[0] < 'A' || in[0] > 'J' || in[1] < '0' || in[1] > '9' {
			continue
		}
		x := int(in[0] - 'A')
		y := int(in[1] - '0')
		target := boards[turn]
		target.Fire(x, y)
		if target.AllHit() {
			break Game
		}
		if *devMode {
			continue
		}
		if !target.IsBoat(x, y) {
			turn = 1 - turn // switch players
		}
	}
	s.Clear()
	s.RenderBoard(&p1, 0, 0, true)
	s.RenderBoard(&p2, 40, 0, true)
	s.Print()
}

func randomPlacement() string {
	return fmt.Sprintf("%c%v%c",
		"ABCDEFGHIJ"[rand.Intn(10)],
		rand.Intn(10),
		"RD"[rand.Intn(2)])
}
