package math

import (
	"image"
	"image/draw"
)

func cropInline(imgIn image.Image) image.Image {
	b := imgIn.Bounds()
	imgOut := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(imgOut, imgOut.Bounds(), imgIn, b.Min, draw.Src)

	// find the marker on the left
	y0 := 0
	for {
		if imgOut.Pix[y0*imgOut.Stride+3] != 0 {
			break
		}
		y0++
	}
	y1 := y0
	for {
		if imgOut.Pix[(y1+1)*imgOut.Stride+3] == 0 {
			break
		}
		y1++
	}
	yMid := (y0 + y1) / 2

	// find the width of the marker
	xMin := 0
	for imgOut.Pix[imgOut.PixOffset(xMin, yMid)+3] != 0 {
		xMin++
	}

	// find the top-most row of pixels used
	idx := 0
	for imgOut.Pix[idx+3] == 0 {
		idx += 4
	}
	yMin := idx / imgOut.Stride

	// find the bottom-most row of pixels used
	idx = imgOut.Rect.Max.Y*imgOut.Stride - 4
	for imgOut.Pix[idx+3] == 0 {
		idx -= 4
	}
	yMax := idx/imgOut.Stride + 1

	// Centre the crop window vertically.
	if y0-yMin > yMax-1-y1 {
		yMax = y0 + y1 - yMin + 1
	} else {
		yMin = y0 + y1 - yMax + 1
	}

	// crop left
leftLoop:
	for xMin < imgOut.Rect.Max.X {
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMin, y)
			if imgOut.Pix[idx+3] != 0 {
				break leftLoop
			}
		}
		xMin++
	}

	// crop right
	xMax := imgOut.Rect.Max.X
rightLoop:
	for xMax > xMin {
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMax-1, y)
			if imgOut.Pix[idx+3] != 0 {
				break rightLoop
			}
		}
		xMax--
	}

	crop := image.Rectangle{
		Min: image.Point{xMin, yMin},
		Max: image.Point{xMax, yMax},
	}
	return imgOut.SubImage(crop)
}

func cropDisplayed(imgIn image.Image) image.Image {
	b := imgIn.Bounds()
	imgOut := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(imgOut, imgOut.Bounds(), imgIn, b.Min, draw.Src)

	// find the top-most row of pixels used
	idx := 0
	for imgOut.Pix[idx+3] == 0 {
		idx += 4
	}
	yMin := idx / imgOut.Stride

	// find the bottom-most row of pixels used
	idx = imgOut.Rect.Max.Y*imgOut.Stride - 4
	for imgOut.Pix[idx+3] == 0 {
		idx -= 4
	}
	yMax := idx/imgOut.Stride + 1

	// crop left and right
	xMin := 0
	xMax := imgOut.Rect.Max.X
leftRightLoop:
	for xMin < imgOut.Rect.Max.X {
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMin, y)
			if imgOut.Pix[idx+3] != 0 {
				break leftRightLoop
			}
		}
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMax-1, y)
			if imgOut.Pix[idx+3] != 0 {
				break leftRightLoop
			}
		}
		xMin++
		xMax--
	}

	crop := image.Rectangle{
		Min: image.Point{xMin, yMin},
		Max: image.Point{xMax, yMax},
	}
	return imgOut.SubImage(crop)
}
