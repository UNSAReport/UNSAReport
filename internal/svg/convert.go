package svg

import (
	"image/png"
	"io"
	"os"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// ConvertSVGToPNG converts an SVG file to PNG at the specified output path.
// The density parameter controls the output resolution in DPI (e.g., 300 for print quality).
func ConvertSVGToPNG(svgPath string, pngPath string, density float64) error {
	// Parse SVG file
	f, err := os.Open(svgPath)
	if err != nil {
		return err
	}
	defer f.Close()

	c, err := canvas.ParseSVG(f)
	if err != nil {
		return err
	}

	// Convert DPI to DPMM (dots per millimeter)
	// 1 inch = 25.4mm, so DPI / 25.4 = DPMM
	resolution := canvas.DPMM(density / 25.4)

	// Rasterize and write to PNG
	return writePNG(pngPath, c, resolution)
}

// ConvertSVGToPNGReader converts an SVG from a reader to PNG at the specified output path.
func ConvertSVGToPNGReader(svgReader io.Reader, pngPath string, density float64) error {
	// Parse SVG from reader
	c, err := canvas.ParseSVG(svgReader)
	if err != nil {
		return err
	}

	// Convert DPI to DPMM
	resolution := canvas.DPMM(density / 25.4)

	// Rasterize and write to PNG
	return writePNG(pngPath, c, resolution)
}

func writePNG(filename string, c *canvas.Canvas, resolution canvas.Resolution) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Rasterize the canvas to an image
	img := rasterizer.Draw(c, resolution, canvas.DefaultColorSpace)

	// Encode as PNG
	encoder := &png.Encoder{}
	return encoder.Encode(out, img)
}
