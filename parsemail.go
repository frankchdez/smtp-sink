package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flashmob/go-guerrilla/backends"
	"github.com/jhillyerd/enmime"
)

func pickFileName(dir, fileName string) (string, error) {
	path := filepath.Join(dir, fileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}

	ext := filepath.Ext(fileName)
	prefix := strings.TrimSuffix(fileName, ext)

	for i := 0; i < 100; i++ {
		fname := fmt.Sprintf("%s_%d%s", prefix, i, ext)
		path := filepath.Join(dir, fname)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		}
	}

	fname := fmt.Sprintf("%s_%s%s", prefix, time.Now().Format("20060102_150405"), ext)
	path = filepath.Join(dir, fname)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}

	return "", fmt.Errorf("Unable to create a unique file for %v in 100 tries", fileName)
}

func saveFile(dir, fileName string, content []byte) error {
	path, err := pickFileName(dir, fileName)
	if err != nil {
		backends.Log().Error(err)
		return err
	}

	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		backends.Log().WithError(err).Info("could not create file", path)
		return err
	}

	n, err := file.Write(content)
	if err != nil {
		backends.Log().WithError(err).Info("could not write to file", path)
		return err
	}

	backends.Log().Debugf("wrote %d bytes\n", n)
	return nil
}

func parseMail(reader io.Reader, dir string) error {
	// Parse message body with enmime.
	env, err := enmime.ReadEnvelope(reader)
	if err != nil {
		return err
	}

	// // mime.Inlines is a slice of inlined attacments.
	// fmt.Printf("Inlines: %v\n", len(env.Inlines))

	// mime.Attachments contains the non-inline attachments.
	backends.Log().Debugf("Attachments: %v", len(env.Attachments))

	for _, a := range env.Attachments {
		backends.Log().Debugf("Attachment: %v, Content-Type: %v", a.FileName, a.ContentType)

		if err := saveFile(dir, a.FileName, a.Content); err != nil {
			return err
		}
	}

	return nil
}
