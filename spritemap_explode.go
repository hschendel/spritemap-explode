package main

import (
	"image"
	"flag"
	"strconv"
	"fmt"
	"os"
	"image/png"
	_ "image/jpeg"
	_ "image/gif"
	"strings"
	"math"
	"path"
)

type SpriteMap interface {
	image.Image
	SubImage(r image.Rectangle) image.Image
}

func imageEmpty(img image.Image) bool {
	for x := 0; x < img.Bounds().Max.X; x++ {
		for y := 0; y < img.Bounds().Max.Y; y++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				return false
			}
		}
	}
	return true
}

func imageMirrorY(img image.Image) image.Image {
	mirrorImg := image.NewNRGBA(img.Bounds())
	mx := img.Bounds().Max.X
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		mx--
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			mirrorImg.Set(mx, y, img.At(x, y))
		}
	}
	return mirrorImg
}

type args struct {
	Filename    string
	Prefix      string
	Suffix      string
	FrameWidth  uint
	FrameHeight uint
	Columns     uint
	Rows        uint
	MirrorLeft  bool
}

func (a *args) ImageColumns(img SpriteMap) int {
	if a.Columns != 0 {
		return int(a.Columns)
	}
	return img.Bounds().Max.X / int(a.FrameWidth)
}

func (a *args) ImageRows(img SpriteMap) int {
	if a.Rows != 0 {
		return int(a.Rows)
	}
	return img.Bounds().Max.Y / int(a.FrameHeight)
}

func (a *args) ImageFrameWidth(img SpriteMap) int {
	if a.FrameWidth != 0 {
		return int(a.FrameWidth)
	}
	return img.Bounds().Max.X / int(a.Columns)
}

func (a *args) ImageFrameHeight(img SpriteMap) int {
	if a.FrameHeight != 0 {
		return int(a.FrameHeight)
	}
	return img.Bounds().Max.Y / int(a.Rows)
}

func (a *args) FrameFilenameFormat(img SpriteMap) string {
	xDigits := int(math.Ceil(math.Log10(float64(a.ImageColumns(img)))))
	yDigits := int(math.Ceil(math.Log10(float64(a.ImageRows(img)))))

	format := "-%0" + strconv.Itoa(yDigits) + "d-%0" + strconv.Itoa(xDigits) + "d.png"
	if a.MirrorLeft {
		format = "-%s" + format
	}
	format = "%s" + format
	return format
}

func (a *args) parse() bool {
	flag.UintVar(&a.FrameWidth, "width", 0, "Frame width of one sprite")
	flag.UintVar(&a.FrameHeight, "height", 0, "Frame height of one sprite")
	flag.UintVar(&a.Columns, "columns", 0, "Fumber of columns. Frame width is calculated by dividing the source image width by this number.")
	flag.UintVar(&a.Rows, "rows", 0, "Fumber of rows. Frame height is calculated by dividing the source image height by this number.")
	flag.BoolVar(&a.MirrorLeft, "mirror-left", false, "Every frame is duplicated and flipped on the y axis, i.e. facing left if it has been facing right before."+
		" The file name scheme is then extended to <prefix>-<l|r>-<row index>-<column index> with r being the original.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <filename>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s creates files for each frame in a sprite map. The new files will be named\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "using the scheme <prefix>-<row index>-<column index>.png. Empty frames will be")
		fmt.Fprintln(os.Stderr, "omitted. The rows and columns are counted starting with 0.\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return false
	}
	a.Filename = flag.Arg(0)
	a.Suffix = path.Ext(a.Filename)
	a.Prefix = strings.TrimSuffix(a.Filename, a.Suffix)

	if a.FrameHeight == 0 && a.Rows == 0 {
		os.Stderr.WriteString("Need to set either -height or -rows\n")
		flag.Usage()
		return false
	}

	if a.FrameWidth == 0 && a.Columns == 0 {
		os.Stderr.WriteString("Need to set either -width or -columns\n")
		return false
	}

	return true
}

func saveImage(img image.Image, filename string) {
	file, createErr := os.Create(filename)
	if createErr != nil {
		fmt.Fprintln(os.Stderr, "Cannot create file", filename + ":", createErr)
		return
	}
	encodeErr := png.Encode(file, img)
	file.Close()
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, "Cannot encode image into", filename + ":", encodeErr)
		os.Remove(filename)
	}
}

func explode(a *args, img SpriteMap) {
	frameWidth := a.ImageFrameWidth(img)
	frameHeight := a.ImageFrameHeight(img)
	columns := a.ImageColumns(img)
	rows := a.ImageRows(img)
	format := a.FrameFilenameFormat(img)

	for row := 0; row < rows; row++ {
		y := row * frameHeight
		for column := 0; column < columns ; column++ {
			x := column * frameWidth
			subImage := img.SubImage(image.Rect(x, y, x + frameWidth, y + frameHeight))
			if imageEmpty(subImage){
				continue
			}

			if a.MirrorLeft {
				filenameR := fmt.Sprintf(format, a.Prefix, "r", row, column)
				saveImage(subImage, filenameR)
				filenameL := fmt.Sprintf(format, a.Prefix, "l", row, column)
				mirrorImage := imageMirrorY(subImage)
				saveImage(mirrorImage, filenameL)

			} else {
				filename := fmt.Sprintf(format, a.Prefix, row, column)
				saveImage(subImage, filename)
			}
		}
	}
}

func main() {
	var args args
	if !args.parse() {
		os.Exit(1)
	}

	file, openErr := os.Open(args.Filename)
	if openErr != nil {
		fmt.Fprintln(os.Stderr, "Cannot open", args.Filename + ":", openErr)
		os.Exit(2)
	}
	defer file.Close()

	img, imageFormat, decodeErr := image.Decode(file)
	if decodeErr != nil {
		fmt.Fprintln(os.Stderr, "Cannot decode", args.Filename + ":", decodeErr)
		os.Exit(3)
	}

	spriteMap := img.(SpriteMap)
	if spriteMap == nil {
		fmt.Fprintf(os.Stderr,"Image format %s does not support extracting sub-images\n", imageFormat)
		os.Exit(4)
	}

	explode(&args, spriteMap)
}
