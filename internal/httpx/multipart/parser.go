package multipart

import (
	"errors"
	"net/http"

	"komecore/internal/apperror"
)

func ParseSingle(
	r *http.Request,
	field string,
	maxMemory int64,
) (*File, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, err
	}

	src, header, err := r.FormFile(field)
	if err != nil {
		return nil, errors.New("multipart file not found")
	}

	file := File{
		File:        src,
		Filename:    header.Filename,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
	}

	return &file, nil
}

func ParseMultiple(
	r *http.Request,
	field string,
	maxMemory int64,
) ([]File, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, apperror.NewBadRequest(ErrInvalidMultipartForm.Error())
	}

	headers := r.MultipartForm.File[field]
	if len(headers) == 0 {
		return nil, apperror.NewNotFound(ErrNotFoundMultipartFiles.Error())
	}

	files := make([]File, 0, len(headers))
	for _, h := range headers {
		src, err := h.Open()
		if err != nil {
			return nil, err
		}

		files = append(files, File{
			File:        src,
			Filename:    h.Filename,
			Size:        h.Size,
			ContentType: h.Header.Get("Content-Type"),
		})
	}

	return files, nil
}
