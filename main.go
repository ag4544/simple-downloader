package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func downloadFile(ctx context.Context, url string, filepath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for %s: %v", url, err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request error %s: %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response error %s : %s", url, resp.Status)
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("file creation error %s: %v", filepath, err)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error during file download %s: %v", filepath, err)
	}

	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	urlfile, err := os.Open("url.txt")
	if err != nil {
		fmt.Printf("url list file opening error: %v\n", err)
		return
	}
	defer urlfile.Close()

	scanner := bufio.NewScanner(urlfile)
	var urls []string
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("url list file reading error: %v\n", err)
		return
	}

	var g errgroup.Group
	var errorMessages []string

	var mu sync.Mutex
	if _, err := os.Stat("downloads"); os.IsNotExist(err) {
		if err := os.Mkdir("downloads", os.ModePerm); err != nil {
			fmt.Printf("error creating download dir: %v\n", err)
			return
		}
	}
	for _, url := range urls {
		url := url

		g.Go(func() error {
			fileName := path.Base(url)
			filePath := fmt.Sprintf("./downloads/%s", fileName)
			if err := downloadFile(ctx, url, filePath); err != nil {
				mu.Lock()
				errorMessages = append(errorMessages, err.Error())
				mu.Unlock()
				return err
			}
			fmt.Printf("file downloaded successfully: %s\n", fileName)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		for _, errorMessage := range errorMessages {
			fmt.Printf("an error has occured: %v\n", errorMessage)
		}
	} else {
		fmt.Println("Download complete")
	}
}
