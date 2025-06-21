package profile

import (
	"archive/zip"
	"io"

	"gopkg.in/yaml.v3"
)

func parseExportR2X(file *zip.File) (*ExportR2X, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var exportR2X ExportR2X
	err = yaml.Unmarshal(data, &exportR2X)
	if err != nil {
		return nil, err
	}

	return &exportR2X, nil
}
