package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type Downloader struct {
	concurrency int
	resume      bool

	bar *progressbar.ProgressBar
}

func NewDownloader(concurrency int, resume bool) *Downloader {
	return &Downloader{concurrency: concurrency, resume: resume}
}

func (d *Downloader) Download(strURL, outputpath string, silence bool) error {
	if outputpath == "" {
		outputpath = path.Base(strURL)
	}

	resp, err := http.Head(strURL)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		return d.multiDownload(strURL, outputpath, int(resp.ContentLength), silence)
	}

	return d.singleDownload(strURL, outputpath, silence)
}

func (d *Downloader) multiDownload(strURL, outputpath string, contentLen int, silence bool) error {
	if !silence {
		d.setBar(contentLen)
	}

	partSize := contentLen / d.concurrency

	b, err := d.PathExists(outputpath)
	if err != nil {
		fmt.Printf("PathExists(%s),err(%v)\n", outputpath, err)
		return err
	}
	if b {
		os.RemoveAll(outputpath)
	}
	os.Mkdir(outputpath, 0777)

	var wg sync.WaitGroup
	wg.Add(d.concurrency)

	rangeStart := 0

	for i := 0; i < d.concurrency; i++ {
		go func(i, rangeStart int) {
			defer wg.Done()

			rangeEnd := rangeStart + partSize
			if i == d.concurrency-1 {
				rangeEnd = contentLen
			}

			downloaded := 0
			if d.resume {
				partFileName := d.getPartFilename(strURL, outputpath, i)
				content, err := os.ReadFile(partFileName)
				if err == nil {
					downloaded = len(content)
				}
				if !silence {
					d.bar.Add(downloaded)
				}
			}

			d.downloadPartial(strURL, outputpath, rangeStart+downloaded, rangeEnd, i, silence)

		}(i, rangeStart)

		rangeStart += partSize + 1
	}

	wg.Wait()

	d.merge(strURL, outputpath)

	return nil
}

func (d *Downloader) downloadPartial(strURL, outputpath string, rangeStart, rangeEnd, i int, silence bool) {
	if rangeStart >= rangeEnd {
		return
	}

	req, err := http.NewRequest("GET", strURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if d.resume {
		flags |= os.O_APPEND
	}

	partFile, err := os.OpenFile(d.getPartFilename(strURL, outputpath, i), flags, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer partFile.Close()

	buf := make([]byte, 32*1024)
	if !silence {
		_, err = io.CopyBuffer(io.MultiWriter(partFile, d.bar), resp.Body, buf)
	} else {
		_, err = io.CopyBuffer(partFile, resp.Body, buf)
	}
	if err != nil {
		if err == io.EOF {
			return
		}
		log.Fatal(err)
	}
}

func (d *Downloader) merge(strURL, outputpath string) error {
	destFile, err := os.OpenFile(path.Join(outputpath, path.Base(strURL)), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer destFile.Close()

	for i := 0; i < d.concurrency; i++ {
		partFileName := d.getPartFilename(strURL, outputpath, i)
		partFile, err := os.Open(partFileName)
		if err != nil {
			return err
		}
		io.Copy(destFile, partFile)
		partFile.Close()
		os.Remove(partFileName)
	}

	return nil
}

func (d *Downloader) PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (d *Downloader) getPartFilename(strURL, outputpath string, partNum int) string {
	return fmt.Sprintf("%s-%d", path.Join(outputpath, path.Base(strURL)), partNum)
}

func (d *Downloader) singleDownload(strURL, outputpath string, silence bool) error {
	resp, err := http.Get(strURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !silence {
		d.setBar(int(resp.ContentLength))
	}

	f, err := os.OpenFile(path.Join(outputpath, path.Base(strURL)), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)

	if !silence {
		_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
	} else {
		_, err = io.CopyBuffer(f, resp.Body, buf)
	}
	return err
}

func (d *Downloader) setBar(length int) {
	d.bar = progressbar.NewOptions(
		length,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}
