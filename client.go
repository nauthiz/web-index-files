package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const (
	EntryTypeDirectory = iota
	EntryTypeFile
)

type EntryType int

type Credential struct {
	UserName string
	Password string
}

type WebIndexClient struct {
	httpClient *http.Client
	credential *Credential
}

func (c *WebIndexClient) NewRequest(method string, url string, body io.Reader) (request *http.Request, err error) {
	request, err = http.NewRequest(method, url, body)

	if err != nil {
		return
	}

	if c.credential != nil {
		request.SetBasicAuth(c.credential.UserName, c.credential.Password)
	}

	return
}

func (c *WebIndexClient) WalkEntries(baseUrl string, handler func(baseUrl string, entryType EntryType, entryUrl string) error) (err error) {
	request, err := c.NewRequest("GET", baseUrl, nil)

	if err != nil {
		return
	}

	res, err := c.httpClient.Do(request)

	if err != nil {
		return
	}

	defer func() {
		closeErr := res.Body.Close()

		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)

	if err != nil {
		return
	}

	var handlerErr error = nil

	doc.Find("body > pre > a").Each(func(i int, s *goquery.Selection) {
		entryPath, ok := s.Attr("href")
		if !ok {
			return
		}

		if entryPath == "../" {
			return
		}

		var entryType EntryType

		if strings.HasSuffix(entryPath, "/") {
			entryType = EntryTypeDirectory
			entryPath = strings.TrimSuffix(entryPath, "/")
		} else {
			entryType = EntryTypeFile
		}

		if err := handler(baseUrl, entryType, entryPath); err != nil {
			handlerErr = err
			return
		}
	})

	return handlerErr
}

func (c *WebIndexClient) PrintEntries(url string) (err error) {
	printer := &entryPrinter{Client: c, BaseUrl: url}
	return c.WalkEntries(url, printer.printEntry)
}

func (c *WebIndexClient) DownloadEntries(url string, outputPath string) (err error) {
	downloader := &entryDownloader{Client: c, BaseUrl: url, OutputDirectoryPath: outputPath}
	return c.WalkEntries(url, downloader.downloadEntry)
}

func (c *WebIndexClient) DownloadEntryFile(entryUrl string, outputPath string) (err error) {
	request, err := c.NewRequest("GET", entryUrl, nil)

	if err != nil {
		return
	}

	response, err := c.httpClient.Do(request)

	if err != nil {
		return
	}

	defer func() {
		closeErr := response.Body.Close()

		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if response.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", response.StatusCode, response.Status)
	}

	bytes, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	err = ioutil.WriteFile(outputPath, bytes, 0644)

	if err != nil {
		return
	}

	return nil
}

func makeHttpClient() *http.Client {
	return &http.Client{Timeout: time.Duration(30) * time.Second}
}

type entryDownloader struct {
	Client              *WebIndexClient
	BaseUrl             string
	OutputDirectoryPath string
}

func (d *entryDownloader) downloadEntry(baseUrl string, entryType EntryType, entryPath string) (err error) {
	var directoryPath string

	if baseUrl == d.BaseUrl {
		directoryPath = ""
	} else if strings.HasPrefix(baseUrl, d.BaseUrl+"/") {
		directoryPath = strings.TrimPrefix(baseUrl, d.BaseUrl+"/")
	} else {
		return fmt.Errorf("entry file url must be start with \"%v\"", d.BaseUrl+"/")
	}

	outputEntryPath := path.Join(d.OutputDirectoryPath, directoryPath, entryPath)

	if entryType == EntryTypeDirectory {
		_, statErr := os.Stat(outputEntryPath)

		if statErr != nil {
			err = os.Mkdir(outputEntryPath, 0777)

			if err != nil {
				return
			}
		}

		fmt.Printf("Creating Directory ... %v\n", path.Join(directoryPath, entryPath)+"/")
		return d.Client.WalkEntries(baseUrl+"/"+entryPath, d.downloadEntry)
	} else if entryType == EntryTypeFile {
		fmt.Printf("Downloading File ...   %v\n", path.Join(directoryPath, entryPath))
		return d.Client.DownloadEntryFile(baseUrl+"/"+entryPath, outputEntryPath)
	} else {
		return fmt.Errorf("\"%v\" unsupported entry type ", entryType)
	}
}

type entryPrinter struct {
	Client  *WebIndexClient
	BaseUrl string
}

func (p *entryPrinter) printEntry(baseUrl string, entryType EntryType, entryPath string) (err error) {
	var directoryPath string

	if baseUrl == p.BaseUrl {
		directoryPath = ""
	} else if strings.HasPrefix(baseUrl, p.BaseUrl+"/") {
		directoryPath = strings.TrimPrefix(baseUrl, p.BaseUrl+"/")
	} else {
		return fmt.Errorf("entry file url must be start with \"%v\"", p.BaseUrl+"/")
	}

	fullEntryPath := path.Join(directoryPath, entryPath)

	if entryType == EntryTypeDirectory {
		_, statErr := os.Stat(fullEntryPath)

		if statErr != nil {
			err = os.Mkdir(fullEntryPath, 0777)

			if err != nil {
				return
			}
		}

		fmt.Println(fullEntryPath + "/")
		return p.Client.WalkEntries(baseUrl+"/"+entryPath, p.printEntry)
	} else if entryType == EntryTypeFile {
		fmt.Println(fullEntryPath)
		return nil
	} else {
		return fmt.Errorf("\"%v\" unsupported entry type ", entryType)
	}
}
