package wire

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// font5x7 draws the digits 0 to 9, five columns by seven rows, a row a string.
var font5x7 = [10][7]string{
	{".###.", "#...#", "#..##", "#.#.#", "##..#", "#...#", ".###."},
	{"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	{".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	{"#####", "...#.", "..#..", "...#.", "....#", "#...#", ".###."},
	{"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	{"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	{"..##.", ".#...", "#....", "####.", "#...#", "#...#", ".###."},
	{"#####", "....#", "...#.", "..#..", ".#...", ".#...", ".#..."},
	{".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	{".###.", "#...#", "#...#", ".####", "....#", "...#.", ".##.."},
}

// DigitsPNG is a picture of digits, black on white, large enough for any model that reads
// pictures: the vision tier's gate (SB-6101) asks it to read them back.
func DigitsPNG(digits string) []byte {
	const scale, gap = 12, 2
	w := (len(digits)*(5+gap) + gap) * scale
	h := (7 + 2*gap) * scale
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	for n, d := range digits {
		if d < '0' || d > '9' {
			continue
		}
		g := font5x7[d-'0']
		for y, row := range g {
			for x, c := range row {
				if c != '#' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetGray((gap+n*(5+gap)+x)*scale+dx, (gap+y)*scale+dy, color.Gray{})
					}
				}
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
