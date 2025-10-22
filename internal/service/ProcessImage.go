package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/zlog"
	xdraw "golang.org/x/image/draw"

	"ImageProcessor/internal/model"
	"ImageProcessor/internal/repository"
)

var (
	ErrBadParameters = fmt.Errorf("bad parameters")
	ErrUnknowMode    = fmt.Errorf("unknow mode")
)

var (
	ThumbnailsHeight = 150
	ThumbnailsWidth  = 150
)

type ImageService struct {
	Ctx          context.Context
	ImageStorage repository.ImageStore
	Img          model.ImageTask
}

func ProcessImage(is ImageService) (model.ImageInRepo, error) {
	switch is.Img.TypeProcessing {
	case "watermark":
		return watermark(is)
	case "resize":
		return resize(is)
	case "thumbnail":
		is.Img.Parameters.Height = &ThumbnailsHeight
		is.Img.Parameters.Width = &ThumbnailsWidth
		return resize(is)
	default:
		return model.ImageInRepo{}, ErrUnknowMode
	}
}

func resize(is ImageService) (model.ImageInRepo, error) {
	baseFile, err := is.ImageStorage.Download(is.Ctx, is.Img.UploadsPath)
	if err != nil {
		return model.ImageInRepo{}, err
	}
	defer func() {
		if err := baseFile.Close(); err != nil {
			zlog.Logger.Error().Msg(err.Error())
		}
	}()

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(baseFile)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	gifReader := bytes.NewReader(buf.Bytes())

	gifData, err := gif.DecodeAll(gifReader)
	if err == nil {
		return resizeGIF(is, gifData)
	}

	imageReader := bytes.NewReader(buf.Bytes())
	baseImg, format, err := image.Decode(imageReader)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	baseBounds := baseImg.Bounds()
	if is.Img.Parameters.Height == nil || is.Img.Parameters.Width == nil {
		return model.ImageInRepo{}, ErrBadParameters
	}

	newWidth := *is.Img.Parameters.Width
	newHeight := *is.Img.Parameters.Height
	resizedImage := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	xdraw.BiLinear.Scale(resizedImage, resizedImage.Bounds(), baseImg, baseBounds.Bounds(), draw.Over, nil)

	var outFileName string

	switch is.Img.TypeProcessing {
	case "resize":
		outFileName = fmt.Sprintf("processed/resized/%s-resized.%s", uuid.New().String(), format)
	case "thumbnail":
		outFileName = fmt.Sprintf("processed/thumbnails/%s-thumbnails.%s", uuid.New().String(), format)
	}

	err = saveImage(is, outFileName, resizedImage, format)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	return model.ImageInRepo{
		ID:            is.Img.ImageID,
		UploadsPath:   is.Img.UploadsPath,
		ProcessedPath: outFileName,
		Processed:     true,
	}, nil
}

func resizeGIF(is ImageService, gifData *gif.GIF) (model.ImageInRepo, error) {
	if is.Img.Parameters.Height == nil || is.Img.Parameters.Width == nil {
		return model.ImageInRepo{}, ErrBadParameters
	}

	newWidth := *is.Img.Parameters.Width
	newHeight := *is.Img.Parameters.Height

	resizedGIF := &gif.GIF{
		Image:    make([]*image.Paletted, len(gifData.Image)),
		Delay:    gifData.Delay,
		Disposal: gifData.Disposal,
	}

	for i, frame := range gifData.Image {
		bounds := frame.Bounds()

		rgbaFrame := image.NewRGBA(bounds)
		draw.Draw(rgbaFrame, bounds, frame, bounds.Min, draw.Src)

		resizedFrame := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		xdraw.BiLinear.Scale(resizedFrame, resizedFrame.Bounds(), rgbaFrame, bounds, draw.Over, nil)

		palettedFrame := image.NewPaletted(resizedFrame.Bounds(), frame.Palette)
		draw.Draw(palettedFrame, palettedFrame.Bounds(), resizedFrame, resizedFrame.Bounds().Min, draw.Src)

		resizedGIF.Image[i] = palettedFrame
	}

	var outFileName string
	switch is.Img.TypeProcessing {
	case "resize":
		outFileName = fmt.Sprintf("processed/resized/%s-resized.gif", uuid.New().String())
	case "thumbnail":
		outFileName = fmt.Sprintf("processed/thumbnails/%s-thumbnails.gif", uuid.New().String())
	}

	err := saveGIF(is, outFileName, resizedGIF)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	return model.ImageInRepo{
		ID:            is.Img.ImageID,
		UploadsPath:   is.Img.UploadsPath,
		ProcessedPath: outFileName,
		Processed:     true,
	}, nil
}

func watermark(is ImageService) (model.ImageInRepo, error) {
	baseFile, err := is.ImageStorage.Download(is.Ctx, is.Img.UploadsPath)
	if err != nil {
		return model.ImageInRepo{}, err
	}
	defer func() {
		if err := baseFile.Close(); err != nil {
			zlog.Logger.Error().Msg(err.Error())
		}
	}()

	baseImg, format, err := image.Decode(baseFile)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	overlayFile, err := is.ImageStorage.Download(is.Ctx, *is.Img.Parameters.WatermarkPath)
	if err != nil {
		return model.ImageInRepo{}, err
	}
	defer func() {
		if err := overlayFile.Close(); err != nil {
			zlog.Logger.Error().Msg(err.Error())
		}
	}()

	overlayImg, _, err := image.Decode(overlayFile)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	baseBounds := baseImg.Bounds()

	newWidth := baseBounds.Dx()
	newHeight := baseBounds.Dy()
	resizedOverlay := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	xdraw.BiLinear.Scale(resizedOverlay, resizedOverlay.Bounds(), overlayImg, overlayImg.Bounds(), draw.Over, nil)

	output := image.NewRGBA(baseBounds)
	draw.Draw(output, baseBounds, baseImg, image.Point{0, 0}, draw.Src)
	draw.Draw(output, resizedOverlay.Bounds(), resizedOverlay, image.Point{}, draw.Over)

	outFileName := fmt.Sprintf("processed/watermarked/%s-watermarked.%s", uuid.New().String(), format)

	err = saveImage(is, outFileName, output, format)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	err = is.ImageStorage.Delete(is.Ctx, *is.Img.Parameters.WatermarkPath)
	if err != nil {
		zlog.Logger.Error().Msg(err.Error())
	}

	return model.ImageInRepo{
		ID:            is.Img.ImageID,
		UploadsPath:   is.Img.UploadsPath,
		ProcessedPath: outFileName,
		Processed:     true,
	}, nil
}

func saveImage(is ImageService, outFilename string, img image.Image, format string) error {
	outFile := &bytes.Buffer{}
	fileWriter := bufio.NewWriter(outFile)

	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(fileWriter, img, &jpeg.Options{Quality: 90})
	case "gif":
		err = gif.Encode(fileWriter, img, nil)
	case "png":
		err = png.Encode(fileWriter, img)
	default:
		err = png.Encode(fileWriter, img)
	}

	if err != nil {
		return err
	}

	fileReader := bufio.NewReader(outFile)
	err = is.ImageStorage.Upload(is.Ctx, fileReader, outFilename, int64(outFile.Len()))
	if err != nil {
		return err
	}
	return nil
}

func saveGIF(is ImageService, outFilename string, gifData *gif.GIF) error {
	outFile := &bytes.Buffer{}

	err := gif.EncodeAll(outFile, gifData)
	if err != nil {
		return err
	}

	fileReader := bufio.NewReader(outFile)
	err = is.ImageStorage.Upload(is.Ctx, fileReader, outFilename, int64(outFile.Len()))
	if err != nil {
		return err
	}
	return nil
}
