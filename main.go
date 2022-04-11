package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	path := "messages"
	err := filepath.WalkDir(path, func(path string, f os.DirEntry, err error) error {
		if f.Name() == path {
			return nil
		}
		if f.IsDir() && f.Name()[0] != '-' { //directories starts with "-" contains messages with communities/bots
			return iterateOverMessageDir(path, f.Name())
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func iterateOverMessageDir(path, dirName string) error {
	var oggCount, jpgCount int64
	return filepath.WalkDir(path, func(path string, f os.DirEntry, err error) error {
		if f.Name() == dirName {
			return nil
		}
		if strings.Contains(f.Name(), "messages") {
			err = process(path, &oggCount, &jpgCount)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func process(filePath string, oggCount, jpgCount *int64) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	reader, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	dialogueName := reader.Find("div.message__header > a").First().Text()
	err = os.MkdirAll(fmt.Sprintf("files/%s/voices", dialogueName), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(fmt.Sprintf("files/%s/pictures", dialogueName), os.ModePerm)
	if err != nil {
		return err
	}

	reader.Find(".attachment__link").Each(func(i int, selection *goquery.Selection) {
		fileURL, exists := selection.Attr("href")
		if exists {
			fileBytes, err := downloadFile(fileURL)
			if err != nil {
				return
			}

			var fPath string
			switch fExt := fileURL[len(fileURL)-3:]; fExt {
			case "ogg":
				c := atomic.AddInt64(oggCount, 1)
				fPath = fmt.Sprintf("files/%s/voices/%d.ogg", dialogueName, c)
			case "jpg":
				c := atomic.AddInt64(jpgCount, 1)
				fPath = fmt.Sprintf("files/%s/pictures/%d.jpg", dialogueName, c)
			default:
				return
			}
			err = createFile(fileBytes, fPath)
			if err != nil {
				return
			}
		}
	})
	return err
}

func createFile(b *bytes.Buffer, f string) error {
	file, err := os.Create(f)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, bufio.NewReader(b))
	if err != nil {
		return err
	}
	return nil
}

func downloadFile(URL string) (*bytes.Buffer, error) {
	res, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New("received non 200 response code")
	}

	var picBytes bytes.Buffer
	_, err = io.Copy(bufio.NewWriter(&picBytes), res.Body)
	if err != nil {
		return nil, err
	}
	return &picBytes, nil
}
