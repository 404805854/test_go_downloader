package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/urfave/cli/v2"
	"mvdan.cc/xurls/v2"
)

func removeDuplication_map(arr []string) []string {
	set := make(map[string]struct{}, len(arr))

	j := 0
	for _, v := range arr {
		_, ok := set[v]
		if ok {
			continue
		}
		set[v] = struct{}{}
		arr[j] = v
		j++
	}

	return arr[:j]
}

func removeDuplication_sort(arr []string) []string {
	sort.Strings(arr)

	length := len(arr)
	if length == 0 {
		return arr
	}

	j := 0
	for i := 1; i < length; i++ {
		if arr[i] != arr[j] {
			j++
			if j < i {
				swap(arr, i, j)
			}
		}
	}

	return arr[:j+1]
}

func swap(arr []string, a, b int) {
	arr[a], arr[b] = arr[b], arr[a]
}

func download_sync(content, strURL, outputpath string, concurrency int, public, resume, silence bool) string {
	if !strings.HasSuffix(outputpath, "/") {
		if strings.Contains(path.Base(outputpath), ".") {
			outputpath, _ = filepath.Split(outputpath)
			if outputpath == "" {
				outputpath = "./"
			}
		}
	}

	rxStrict := xurls.Strict()

	URLs := removeDuplication_sort(rxStrict.FindAllString(content, -1))
	process_threads := concurrency / len(URLs)
	if process_threads == 0 {
		process_threads = 1
	}
	var wg sync.WaitGroup
	if concurrency < len(URLs)/process_threads {
		wg.Add(concurrency)
	} else {
		wg.Add(len(URLs))
	}
	if len(URLs) > 1 {
		silence = true
	}

	for _, strURL := range URLs {
		go func(strURL, outputpath string, concurrency int, resume, silence bool) {
			defer wg.Done()

			if public && !strings.HasPrefix(strURL, "https://") {
				return
			}
			filename := path.Join(outputpath, strings.Split(path.Base(strURL), ".")[0])
			err := NewDownloader(concurrency, resume).Download(strURL, filename, silence)
			if err != nil {
				log.Fatal(err)
				return
			}
			content = strings.Replace(content, strURL, path.Join(filename, path.Base(strURL)), -1)
			if !silence {
				fmt.Println()
			}
		}(strURL, outputpath, process_threads, resume, silence)
	}

	wg.Wait()
	return content
}

func main() {
	// ???????????????
	concurrencyN := runtime.NumCPU()

	app := &cli.App{
		Name:  "downloader",
		Usage: "File concurrency downloader",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "`URL` to download (If it appears at the same time, this shall prevail)",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "string",
				Aliases: []string{"s"},
				Usage:   "String containing `URL`",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "./",
				Usage:   "Output `filepath`",
			},
			&cli.IntFlag{
				Name:    "concurrency",
				Aliases: []string{"n"},
				Value:   concurrencyN,
				Usage:   "Concurrency `number`",
			},
			&cli.BoolFlag{
				Name:    "resume",
				Aliases: []string{"r"},
				Value:   true,
				Usage:   "Resume download",
			},
			&cli.BoolFlag{
				Name:    "silence",
				Aliases: []string{"sil"},
				Value:   false,
				Usage:   "Silence download",
			},
			&cli.BoolFlag{
				Name:    "public",
				Aliases: []string{"p"},
				Value:   false,
				Usage:   "Simulate public cloud",
			},
		},
		Action: func(c *cli.Context) error {
			var content, outputpath string
			strURL := c.String("url")
			content = c.String("string")
			outputpath = c.String("output")
			concurrency := c.Int("concurrency")
			resume := c.Bool("resume")
			silence := c.Bool("silence")
			public := c.Bool("public")

			if strURL == "" && content == "" {
				return nil
			}
			if strURL != "" {
				content = strURL
			}
			fmt.Println(download_sync(content, strURL, outputpath, concurrency, public, resume, silence))
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
