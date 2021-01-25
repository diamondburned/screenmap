package main

import (
	"io"
	"os"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func readImage(path, typ string, maxRect *gdk.Rectangle) (image *gtk.Image, err error) {
	image, _ = gtk.ImageNew()
	scale := image.GetScaleFactor()

	var f = os.Stdin

	if path != "" && path != "-" {
		f, err = os.Open(path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open file")
		}
	}

	defer f.Close()

	w := maxRect.GetWidth() * scale
	h := maxRect.GetHeight() * scale

	pixbuf, err := readPixbuf(f, typ, w, h)
	if err != nil {
		return nil, err
	}

	// No need to convert to Surface if scale is 1.
	if scale == 1 {
		image.SetFromPixbuf(pixbuf)
		return image, nil
	}

	surface, err := gdk.CairoSurfaceCreateFromPixbuf(pixbuf, scale, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create scaled surface")
	}

	image.SetFromSurface(surface)
	return image, nil
}

func readPixbuf(r io.Reader, typ string, w, h int) (pixbuf *gdk.Pixbuf, err error) {
	var l *gdk.PixbufLoader
	if typ != "" {
		l, err = gdk.PixbufLoaderNewWithType(typ)
	} else {
		l, err = gdk.PixbufLoaderNew()
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to create pixbuf loader")
	}

	defer l.Close()

	l.Connect("size-prepared", func(l *gdk.PixbufLoader) {
		l.SetSize(w, h)
	})

	buf := make([]byte, 4*1024*1024) // use a large 4MB buffer.

	if _, err := io.CopyBuffer(l, r, buf); err != nil {
		return nil, errors.Wrap(err, "pixbuf copy failed")
	}

	pixbuf, err = l.GetPixbuf()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pixbuf")
	}

	return pixbuf, nil
}
