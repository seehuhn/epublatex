package render

import "image"

type BookImageType int

const (
	BookImageTypePNG BookImageType = iota
	BookImageTypeJPG
)

type BookImage struct {
	Key       string
	Image     image.Image
	ImageAttr string
	Name      string
	Folder    string
	Type      BookImageType
}
