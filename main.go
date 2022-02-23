// mojimg -- commandline image file generator with text
//
// preparation:
//		1. Put a ttf file in "font" folder
//         The name of ttf file must end with "mr.ttf" (this is a restriction of draw2d)
//           font/
//             YOURTTFFILEmr.ttf
//           src/
//			   mojimg.go
//
//		2. Do go get dependencies:
//           go get github.com/llgcode/draw2d
//           go get github.com/andrew-d/go-termutil
//
//      3. Do go build src/mojimmg.go
//
// usage:
//      help:
//          mojimg -h
//      simple:
//          mojimg -pos rightbottom -fg #f00 -output hello.png "Hello, 世界"
//      multi-line:
//          mojimg -pos rightbottom -fg #f00 -output hello.png line1 line2 line3
//      multi-position (with pipe):
//          mojimg -pos rightbottom -fg #f00 line1  |  mojimg -pos centermiddle -output hello.png line2
//      overwrite:
//          cat mywallpaper.png | mojimg -pos rightbottom -fg #f00 -output hello.png "Hello, 世界"
//
// TODO: alpha
// TODO: pretty cl options

package main

import (
	"bufio"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"

	"github.com/andrew-d/go-termutil"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"

	"github.com/shu-go/gli"
)

type Chip struct {
	Image *image.RGBA
	X, Y  int
}

type OutputType uint

const (
	PNG OutputType = iota
	JPEG
)

var (
	emojiRE *regexp.Regexp = regexp.MustCompile("(::[^:]+::)")
)

type globalCmd struct {
	Output string `cli:"output=FILE_NAME" help:"generated file name"`
	Type   string `cli:"type=IMAGE_FILE_TYPE" default:"png" help:"png or jpg. for stdout. (if -output is given, -type is set to the extension)"`

	Width  int `cli:"width" default:"1024" help:"image width"`
	Height int `cli:"height" default:"768" help:"image height"`

	Pos string `cli:"pos" default:"topleft" help:"combination of [top | middle | bottom] and [left | center | right] or [t | m | b] and [l | c | r]"`

	FontName string `cli:"font" default:"ipag" help:"ttf file name without suffix \"mr.ttf\""`

	BG string `cli:"bg" default:"#ffff" , help:"#RGBA"`
	FG string `cli:"fg" default:"#000f" , help:"#RGBA"`
}

func (cmd globalCmd) Run(texts []string) error {
	if len(texts) == 0 {
		return fmt.Errorf("no text passed")
	}

	// outputType
	outputType := PNG
	if strings.Contains(cmd.Output, ".") {
		cmd.Type = strings.ToLower(cmd.Output[strings.LastIndex(cmd.Output, ".")+1:])
	}
	switch cmd.Type {
	case "jpg":
		outputType = JPEG
	default:
		outputType = PNG
	}

	// bg, fg
	if strings.HasPrefix(cmd.BG, "#") {
		cmd.BG = cmd.BG[1:]
	}
	if strings.HasPrefix(cmd.FG, "#") {
		cmd.FG = cmd.FG[1:]
	}
	bg, err := parseRGBA(cmd.BG)
	if err != nil {
		return fmt.Errorf("parse BG: %v", err)
	}
	fg, err := parseRGBA(cmd.FG)
	if err != nil {
		return fmt.Errorf("parse FG: %v", err)
	}

	// posv, posh
	posv, posh := parsePosition(cmd.Pos)

	// global settings

	draw2d.SetFontFolder("./font/")

	// construct a rendered image

	var renderedImage *image.RGBA

	if termutil.Isatty(os.Stdin.Fd()) {
		renderedImage = image.NewRGBA(image.Rect(0, 0, cmd.Width, cmd.Height))

		// clear
		if bg.A != 0 {
			gc := draw2dimg.NewGraphicContext(renderedImage)
			gc.SetFillColor(bg)
			gc.Clear()
		}
	} else {
		baseimg, _, err := image.Decode(os.Stdin)
		if err != nil {
			return fmt.Errorf("Failed to load base image from stdin: %v", err)
		}
		b := baseimg.Bounds()
		cmd.Width = b.Max.X - b.Min.X
		cmd.Height = b.Max.Y - b.Min.Y

		renderedImage = image.NewRGBA(image.Rect(0, 0, cmd.Width, cmd.Height))
		draw.Draw(renderedImage, renderedImage.Bounds(), baseimg, image.Point{0, 0}, draw.Src)
	}

	// render

	// chip
	chips := make([]*Chip, 0)
	for _, t := range texts {
		img, err := makeChipImage(t, cmd.FontName, bg, fg)
		if err != nil {
			return err
		}

		chip := &Chip{Image: img, X: 0, Y: 0}

		chips = append(chips, chip)
		//saveImage(fmt.Sprintf("chip%02d.png", i), chip.Image)
	}

	// positioning
	switch posv {
	case "top":
		y := 0
		for _, c := range chips {
			c.Y = y
			y += c.Image.Bounds().Max.Y
		}
	case "middle":
		// measure total height of chips'
		totalHeight := 0
		for _, c := range chips {
			totalHeight += c.Image.Bounds().Max.Y
		}
		// positioning
		y := (cmd.Height - totalHeight) / 2
		for _, c := range chips {
			c.Y = y
			y += c.Image.Bounds().Max.Y
		}
	case "bottom":
		y := cmd.Height
		for i, _ := range chips {
			c := chips[len(chips)-i-1]
			y -= c.Image.Bounds().Max.Y
			c.Y = y
		}
	}
	switch posh {
	case "left":
		// no need to operate
		//for _, c := range chips {
		//	c.X = 0
		//}
	case "center":
		for _, c := range chips {
			b := c.Image.Bounds()
			c.X = (cmd.Width - (b.Max.X - b.Min.X)) / 2
		}
	case "right":
		for _, c := range chips {
			b := c.Image.Bounds()
			c.X = (cmd.Width - (b.Max.X - b.Min.X))
		}
	}

	// pasting chips
	for _, chip := range chips {
		b := chip.Image.Bounds()
		destr := image.Rect(chip.X, chip.Y, chip.X+(b.Max.X-b.Min.X), chip.Y+(b.Max.Y-b.Min.Y))
		draw.Draw(renderedImage, destr, chip.Image, image.Point{0, 0}, draw.Over)
	}

	return saveImage(cmd.Output, outputType, renderedImage)
}

func main() {
	app := gli.NewWith(&globalCmd{})
	app.Version = "0.2.0"
	app.Copyright = "(C) 2018 Shuhei Kubota"
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func rangeOfFoundStringIdxPairs(r [][]int) [][]int {
	result := [][]int{}
	for _, v := range r {
		result = append(result, []int{v[0], v[1]})
	}
	return result
}

func makeChipImage(text, fontname string, bg, fg color.RGBA) (*image.RGBA, error) {
	//
	// determine what to render
	//

	textWOEmojis := ""
	emojiRepos := make(map[string]image.Image)

	m := emojiRE.FindAllStringSubmatchIndex(text, -1)

	if len(m) == 0 {
		textWOEmojis = text
	} else {
		prevEnd := 0
		for _, v := range rangeOfFoundStringIdxPairs(m) {
			textWOEmojis += text[prevEnd:v[0]]
			prevEnd = v[1]
		}
		if prevEnd < len(text)-1 {
			textWOEmojis += text[prevEnd:]
		}

		// download images
		for _, v := range rangeOfFoundStringIdxPairs(m) {
			name := text[v[0]+2 : v[1]-2] // "::smile::" => "smile"
			if _, ok := emojiRepos[name]; ok {
				continue
			}

			resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/arvida/emoji-cheat-sheet.com/master/public/graphics/emojis/%s.png", name))
			if err != nil {
				return nil, fmt.Errorf("Failed to download emoji file of %v: %v", name, err)
			}
			defer resp.Body.Close()

			emojiImg, _, err := image.Decode(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Failed to load image of %v: %v", name, err)
			}
			emojiRepos[name] = emojiImg
		}
	}

	//
	// calc the size of a chip
	//

	// text
	var chipWidth, chipHeight int = 1, 1
	var top, right, bottom float64 = 0, 0, 0
	test := image.NewRGBA(image.Rect(0, 0, 1, 1))
	gc := draw2dimg.NewGraphicContext(test)
	if len(textWOEmojis) != 0 {
		gc.SetFontData(draw2d.FontData{fontname, draw2d.FontFamilyMono, draw2d.FontStyleNormal})
		gc.SetFontSize(48)
		_ /*left*/, top, right, bottom = gc.GetStringBounds(textWOEmojis)
		chipWidth, chipHeight = int(math.Ceil(right)), int(math.Ceil(bottom-top))
	}
	// emojis
	for _, v := range rangeOfFoundStringIdxPairs(m) {
		name := text[v[0]+2 : v[1]-2] // "::smile::" => "smile"
		if emojiImg, ok := emojiRepos[name]; ok {
			b := emojiImg.Bounds()
			ew := b.Max.X - b.Min.X
			eh := b.Max.Y - b.Min.Y

			// height ... max
			if chipHeight < eh {
				chipHeight = eh
			}
			// width ... sum
			chipWidth += ew
		}
	}

	if chipWidth < 0 || chipHeight < 0 {
		return test, nil
	}

	//
	// render
	//

	chipImage := image.NewRGBA(image.Rect(0, 0, chipWidth, chipHeight))
	gc = draw2dimg.NewGraphicContext(chipImage)
	gc.SetFillColor(fg)
	gc.SetFontSize(48)
	gc.SetFontData(draw2d.FontData{fontname, draw2d.FontFamilyMono, draw2d.FontStyleNormal})

	prevEndIdx := 0
	prevEndX := float64(0)
	for _, v := range rangeOfFoundStringIdxPairs(m) {
		// render text before each emoji
		if v[0] != 0 {
			tw := gc.FillStringAt(text[prevEndIdx:v[0]], prevEndX, float64(-top))
			prevEndX += tw
		}

		// render an emoji
		name := text[v[0]+2 : v[1]-2] // "::smile::" => "smile"
		if emojiImg, ok := emojiRepos[name]; ok {
			b := emojiImg.Bounds()
			ew := b.Max.X - b.Min.X
			eh := b.Max.Y - b.Min.Y

			destB := chipImage.Bounds()
			destB.Min.X = int(prevEndX)
			destB.Min.Y = 0
			destB.Max.X = int(prevEndX) + ew
			destB.Max.Y = eh

			draw.Draw(chipImage, destB, emojiImg, image.Point{0, 0}, draw.Src)

			prevEndX += float64(ew)
		} else {
			return nil, fmt.Errorf("Internal inconsistency about emoji %v", name)
		}

		prevEndIdx = v[1]
	}
	if prevEndIdx != len(text)-1 {
		gc.Translate(prevEndX, float64(-top))
		gc.FillString(text[prevEndIdx:])
	}

	return chipImage, nil
}

func parsePosition(pos string) (string, string) {
	var posv, posh string

	if len(pos) > 2 {
		if strings.Contains(pos, "top") {
			posv = "top"
		}
		if strings.Contains(pos, "middle") {
			posv = "middle"
		}
		if strings.Contains(pos, "bottom") {
			posv = "bottom"
		}
		if strings.Contains(pos, "left") {
			posh = "left"
		}
		if strings.Contains(pos, "center") {
			posh = "center"
		}
		if strings.Contains(pos, "right") {
			posh = "right"
		}
	} else {
		if strings.Contains(pos, "t") {
			posv = "top"
		}
		if strings.Contains(pos, "m") {
			posv = "middle"
		}
		if strings.Contains(pos, "b") {
			posv = "bottom"
		}
		if strings.Contains(pos, "l") {
			posh = "left"
		}
		if strings.Contains(pos, "c") {
			posh = "center"
		}
		if strings.Contains(pos, "r") {
			posh = "right"
		}
	}

	return posv, posh
}

func parseRGBA(s string) (color.RGBA, error) {
	var rgba color.RGBA
	var r, g, b uint8
	var a uint8 = 0xff

	var compLen, compRep int
	var hasAlpha bool
	lens := len(s)

	if 3 <= lens && lens <= 4 {
		compLen = 1
		compRep = 2
		if lens == 4 {
			hasAlpha = true
		}
	}

	if 6 <= lens && lens <= 8 {
		compLen = 2
		compRep = 1
		if lens == 8 {
			hasAlpha = true
		}
	}

	p := 0
	compList := []*uint8{&r, &g, &b, &a}
	for compIndex, comp := range compList {
		if compIndex == 3 && !hasAlpha {
			break
		}

		compS := strings.Repeat(s[p:p+compLen], compRep)

		comp64, err := strconv.ParseUint(compS, 16, 8)
		if err != nil {
			return rgba, fmt.Errorf("failed to parse (%v): %v", compS, err)
		}
		*comp = uint8(comp64)

		p += compLen
	}

	rgba = color.RGBA{r, g, b, a}

	return rgba, nil
}

func saveImage(filename string, outputType OutputType, m image.Image) error {
	if len(filename) == 0 {
		b := os.Stdout
		if outputType == JPEG {
			err := jpeg.Encode(b, m, &jpeg.Options{jpeg.DefaultQuality})
			if err != nil {
				return fmt.Errorf("encode: %v", err)
			}
		} else {
			err := png.Encode(b, m)
			if err != nil {
				return fmt.Errorf("encode: %v", err)
			}
		}
	} else {
		filePath := filename
		f, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("create file: %v", err)
		}
		defer f.Close()

		b := bufio.NewWriter(f)

		if outputType == JPEG {
			err := jpeg.Encode(b, m, &jpeg.Options{jpeg.DefaultQuality})
			if err != nil {
				return fmt.Errorf("encode: %v", err)
			}
		} else {
			err = png.Encode(b, m)
			if err != nil {
				return fmt.Errorf("encode: %v", err)
			}
		}
		err = b.Flush()
		if err != nil {
			return fmt.Errorf("flush file: %v", err)
		}
	}

	return nil
}
