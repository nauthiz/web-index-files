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

type DownloadOptions struct {
	Recursive bool
}

type DownloadOption interface {
	ApplyDownloadOption(options *DownloadOptions)
}

type PrintOptions struct {
	Recursive bool
}

type PrintOption interface {
	ApplyPrintOption(options *PrintOptions)
}

type RecursiveOption bool

func (o RecursiveOption) ApplyDownloadOption(options *DownloadOptions) {
	options.Recursive = bool(o)
}

func (o RecursiveOption) ApplyPrintOption(options *PrintOptions) {
	options.Recursive = bool(o)
}

func WithRecursive(recursive bool) RecursiveOption {
	return RecursiveOption(recursive)
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

func (c *WebIndexClient) WalkEntries(url string, handler func(baseUrl string, entryType EntryType, entryUrl string) error) (err error) {
	request, err := c.NewRequest("GET", url, nil)

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

		if err := handler(url, entryType, entryPath); err != nil {
			handlerErr = err
			return
		}
	})

	return handlerErr
}

func (c *WebIndexClient) PrintEntries(url string, options ...PrintOption) (err error) {
	appliedOptions := &PrintOptions{
		Recursive: false,
	}

	for _, option := range options {
		option.ApplyPrintOption(appliedOptions)
	}

	printer := &entryPrinter{Client: c, BaseUrl: url, Recursive: appliedOptions.Recursive}
	return c.WalkEntries(url, printer.printEntry)
}

func (c *WebIndexClient) DownloadEntries(url string, outputPath string, options ...DownloadOption) (err error) {
	appliedOptions := &DownloadOptions{
		Recursive: false,
	}

	for _, option := range options {
		option.ApplyDownloadOption(appliedOptions)
	}

	downloader := &entryDownloader{Client: c, BaseUrl: url, OutputDirectoryPath: outputPath, Recursive: appliedOptions.Recursive}
	return c.WalkEntries(url, downloader.downloadEntry)
}

func (c *WebIndexClient) DownloadEntry(url string, outputPath string) (err error) {
	request, err := c.NewRequest("GET", url, nil)

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
	Recursive           bool
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
			fmt.Printf("Creating Directory ... %v\n", path.Join(directoryPath, entryPath)+"/")
			err = os.Mkdir(outputEntryPath, 0777)

			if err != nil {
				return
			}
		}

		if !d.Recursive {
			return
		}

		return d.Client.WalkEntries(baseUrl+"/"+entryPath, d.downloadEntry)
	} else if entryType == EntryTypeFile {
		fmt.Printf("Downloading File ...   %v\n", path.Join(directoryPath, entryPath))
		return d.Client.DownloadEntry(baseUrl+"/"+entryPath, outputEntryPath)
	} else {
		return fmt.Errorf("\"%v\" unsupported entry type ", entryType)
	}
}

type entryPrinter struct {
	Client  *WebIndexClient
	BaseUrl string
	Recursive bool
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
		fmt.Println(fullEntryPath + "/")

		if !p.Recursive {
			return
		}

		return p.Client.WalkEntries(baseUrl+"/"+entryPath, p.printEntry)
	} else if entryType == EntryTypeFile {
		fmt.Println(fullEntryPath)
		return nil
	} else {
		return fmt.Errorf("\"%v\" unsupported entry type ", entryType)
	}
}
