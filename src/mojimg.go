// mojimg -- commandline image file generator with text
//
// preparation:
//		1. Put a ttf file in "font" folder
//         The name of ttf file must ends with "mr.ttf" (this is a restriction of draw2d)
//           font/
//             YOURTTFFILEmr.ttf
//           src/
//			   mojimg.go
//
//		2. Do go get dependencies:
//           go get code.google.com/p/draw2d/draw2d
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
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"

	"code.google.com/p/draw2d/draw2d"
	"github.com/andrew-d/go-termutil"
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

func main() {
	var output string
	var outputType OutputType
	var fontname string
	var width, height int
	var bg, fg color.RGBA
	var posv, posh string

	// flags parsing

	var outputTypeFlag string
	var posFlag string
	var bgFlag, fgFlag string

	flag.StringVar(&output, "output", "", "generated file name")
	flag.StringVar(&outputTypeFlag, "type", "png", "png or jpg. for stdout. (if -output is given, -type is set to the extension)")
	flag.IntVar(&width, "width", 1024, "image width")
	flag.IntVar(&height, "height", 768, "image height")
	flag.StringVar(&posFlag, "pos", "topleft", "combination of [top | middle | bottom] and [left | center | right] or [t | m | b] and [l | c | r]")
	flag.StringVar(&fontname, "font", "ipag", "ttf file name without suffix \"mr.ttf\"")
	flag.StringVar(&bgFlag, "bg", "#ffff", "#RGBA")
	flag.StringVar(&fgFlag, "fg", "#000f", "#RGBA")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("no text passed")
	}

	// digest into internal values

	// outputType
	if strings.Contains(output, ".") {
		outputTypeFlag = strings.ToLower(output[strings.LastIndex(output, ".")+1:])
	}
	switch outputTypeFlag {
	case "jpg":
		outputType = JPEG
	default:
		outputType = PNG
	}

	// bg, fg
	if strings.HasPrefix(bgFlag, "#") {
		bgFlag = bgFlag[1:]
	}
	if strings.HasPrefix(fgFlag, "#") {
		fgFlag = fgFlag[1:]
	}
	bg, err := parseRGBA(bgFlag)
	if err != nil {
		log.Fatal(err)
	}
	fg, err = parseRGBA(fgFlag)
	if err != nil {
		log.Fatal(err)
	}

	// posv, posh
	posv, posh = parsePosition(posFlag)

	// texts
	texts := append(make([]string, 0), flag.Args()...)

	// global settings

	draw2d.SetFontFolder("./font/")

	// construct a rendered image

	var renderedImage *image.RGBA

	if termutil.Isatty(os.Stdin.Fd()) {
		renderedImage = image.NewRGBA(image.Rect(0, 0, width, height))

		// clear
		if bg.A != 0 {
			gc := draw2d.NewGraphicContext(renderedImage)
			gc.SetFillColor(bg)
			gc.Clear()
		}
	} else {
		baseimg, _, err := image.Decode(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to load base image from stdin: %v", err)
		}
		b := baseimg.Bounds()
		width = b.Max.X - b.Min.X
		height = b.Max.Y - b.Min.Y

		renderedImage = image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(renderedImage, renderedImage.Bounds(), baseimg, image.Point{0, 0}, draw.Src)
	}

	// render

	// chip
	chips := make([]*Chip, 0)
	for _, t := range texts {
		chip := &Chip{Image: makeChipImage(t, fontname, bg, fg), X: 0, Y: 0}
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
		y := (height - totalHeight) / 2
		for _, c := range chips {
			c.Y = y
			y += c.Image.Bounds().Max.Y
		}
	case "bottom":
		y := height
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
			c.X = (width - (b.Max.X - b.Min.X)) / 2
		}
	case "right":
		for _, c := range chips {
			b := c.Image.Bounds()
			c.X = (width - (b.Max.X - b.Min.X))
		}
	}

	// pasting chips
	for _, chip := range chips {
		b := chip.Image.Bounds()
		destr := image.Rect(chip.X, chip.Y, chip.X+(b.Max.X-b.Min.X), chip.Y+(b.Max.Y-b.Min.Y))
		draw.Draw(renderedImage, destr, chip.Image, image.Point{0, 0}, draw.Over)
	}

	saveImage(output, outputType, renderedImage)
}

func makeChipImage(text, fontname string, bg, fg color.RGBA) *image.RGBA {
	test := image.NewRGBA(image.Rect(0, 0, 1, 1))
	gc := draw2d.NewGraphicContext(test)
	gc.SetFontData(draw2d.FontData{fontname, draw2d.FontFamilyMono, draw2d.FontStyleNormal})
	gc.SetFontSize(48)
	_ /*left*/, top, right, bottom := gc.GetStringBounds(text)

	chipWidth, chipHeight := int(math.Ceil(right)), int(math.Ceil(bottom-top))
	if chipWidth < 0 || chipHeight < 0 {
		return test
	}
	//log.Println("chipImage size ", chipWidth, chipHeight)
	chipImage := image.NewRGBA(image.Rect(0, 0, chipWidth, chipHeight))
	gc = draw2d.NewGraphicContext(chipImage)

	gc.SetFillColor(fg)
	gc.SetFontSize(48)
	gc.SetFontData(draw2d.FontData{fontname, draw2d.FontFamilyMono, draw2d.FontStyleNormal})

	var x, y float64
	x, y = 0, -top
	//log.Println("chipImage translate ", x, y)
	gc.Translate(x, y)

	gc.FillString(text)

	return chipImage
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

func saveImage(filename string, outputType OutputType, m image.Image) {
	if len(filename) == 0 {
		b := os.Stdout
		if outputType == JPEG {
			err := jpeg.Encode(b, m, &jpeg.Options{jpeg.DefaultQuality})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err := png.Encode(b, m)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		filePath := filename
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		b := bufio.NewWriter(f)

		if outputType == JPEG {
			err := jpeg.Encode(b, m, &jpeg.Options{jpeg.DefaultQuality})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err = png.Encode(b, m)
			if err != nil {
				log.Fatal(err)
			}
		}
		if err != nil {
			log.Fatal(err)
		}
		err = b.Flush()
		if err != nil {
			log.Fatal(err)
		}
	}
}
