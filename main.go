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
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("os.Executable() error: %s", err.Error())
	}
	executableDirPath := filepath.Dir(ex)
	err = filepath.WalkDir(executableDirPath+"/messages", func(path string, f os.DirEntry, err error) error {
		if f.Name() == path {
			return nil
		}
		if f.IsDir() && !isCommunityChat(f.Name()) && !isGroupChat(f.Name()) { //todo add support of group chats
			return readDialog(path, f.Name(), executableDirPath)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("filepath.WalkDir() error: %s", err.Error())
	}
	log.Println("Done.")
}

func readDialog(path, dirName, executableDir string) error {
	var oggCount, jpgCount int64
	return filepath.WalkDir(path, func(path string, f os.DirEntry, err error) error {
		if f.Name() == dirName {
			return nil
		}
		if strings.Contains(f.Name(), "messages") {
			err = readDialogFile(path, executableDir, &oggCount, &jpgCount)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func readDialogFile(dialogPath, executableDir string, oggCount, jpgCount *int64) error {
	data, err := os.ReadFile(dialogPath)
	if err != nil {
		return err
	}
	reader, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	dialogueName := reader.Find("div.message__header > a").First().Text()
	err = os.MkdirAll(fmt.Sprintf("%s/files/%s/voices", executableDir, dialogueName), os.ModePerm)
	if err != nil {
		log.Printf("mkdir error:%s", err.Error())
		return err
	}

	err = os.MkdirAll(fmt.Sprintf("%s/files/%s/pictures", executableDir, dialogueName), os.ModePerm)
	if err != nil {
		log.Printf("mkdir error:%s", err.Error())
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
				fPath = fmt.Sprintf("%s/files/%s/voices/%d.ogg", executableDir, dialogueName, c) //todo constants
			case "jpg":
				c := atomic.AddInt64(jpgCount, 1)
				fPath = fmt.Sprintf("%s/files/%s/pictures/%d.jpg", executableDir, dialogueName, c)
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

// isCommunityChat used for detecting chats with bots/communities.
// Message directories starts with "-" contains these chats.
func isCommunityChat(s string) bool {
	return s[0] == '-'
}

// isGroupChat used for detecting group chats.
// Message directories starts with "200000" contains these chats.
func isGroupChat(s string) bool {
	return strings.HasPrefix(s, "200000")
}
