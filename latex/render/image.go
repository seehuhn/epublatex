package render

import "image"

type BookImageType int

const (
	BookImageTypePNG BookImageType = iota
	BookImageTypeJPG
)

type BookImage struct {
	Env  string
	Body string

	Alt      string
	CssClass string
	Style    string

	Image image.Image
	Type  BookImageType
}
