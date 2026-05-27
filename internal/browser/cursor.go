package browser

import (
	"math"
	"math/rand"
	"time"

	"github.com/playwright-community/playwright-go"
)

type Point struct {
	X, Y float64
}

// cubicBezier calculates a point on a cubic Bezier curve at time t [0, 1]
func cubicBezier(t float64, p0, p1, p2, p3 Point) Point {
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	x := uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X
	y := uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y

	return Point{x, y}
}

// easeInOutQuad makes the movement start slow and end slow.
func easeInOutQuad(t float64) float64 {
	if t < 0.5 { return 2 * t * t }
	return -1 + (4-2*t)*t
}

// MoveMouseHumanLike moves the Playwright mouse along a curved path to mimic a human.
func MoveMouseHumanLike(mouse playwright.Mouse, start, target Point) error {
	dist := math.Sqrt(math.Pow(target.X-start.X, 2) + math.Pow(target.Y-start.Y, 2))
	if dist < 1 { return nil }

	// Randomized control points for the Bezier curve
	offset := dist / 3.0
	p1 := Point{
		X: start.X + (target.X-start.X)/3 + (rand.Float64()*offset - offset/2),
		Y: start.Y + (target.Y-start.Y)/3 + (rand.Float64()*offset - offset/2),
	}
	p2 := Point{
		X: start.X + 2*(target.X-start.X)/3 + (rand.Float64()*offset - offset/2),
		Y: start.Y + 2*(target.Y-start.Y)/3 + (rand.Float64()*offset - offset/2),
	}

	steps := int(dist/15) + 10 // Dynamic steps based on distance
	if steps > 50 { steps = 50 }

	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		easedT := easeInOutQuad(t)
		pos := cubicBezier(easedT, start, p1, p2, target)

		// Add tiny jitter
		jitterX := rand.Float64()*2 - 1
		jitterY := rand.Float64()*2 - 1

		if err := mouse.Move(pos.X+jitterX, pos.Y+jitterY); err != nil {
			return err
		}

		time.Sleep(time.Duration(rand.Intn(10)+5) * time.Millisecond)
	}

	return mouse.Move(target.X, target.Y)
}
