package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
	"mvdan.cc/xurls/v2"
)

func main() {
	// 默认并发数
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
			if !strings.HasSuffix(outputpath, "/") {
				if strings.Contains(path.Base(outputpath), ".") {
					outputpath, _ = filepath.Split(outputpath)
					if outputpath == "" {
						outputpath = "./"
					}
				}
			}

			rxStrict := xurls.Strict()

			URLs := rxStrict.FindAllString(content, -1)

			for _, strURL := range URLs {
				if public && !strings.HasPrefix(strURL, "https://") {
					continue
				}
				filename := path.Join(outputpath, strings.Split(path.Base(strURL), ".")[0])
				err := NewDownloader(concurrency, resume).Download(strURL, filename, silence)
				if err != nil {
					log.Fatal(err)
					return err
				}
				content = strings.Replace(content, strURL, filename, 1)
				fmt.Println()
			}
			fmt.Println(content)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
