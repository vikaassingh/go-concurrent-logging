package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	// fibGen := FibonacciGenerator()
	// for i := 0; i < 100; i++ {
	// 	fmt.Printf("%d, ", fibGen())
	// }
	StartUrlLog()
}

func FibonacciGenerator() func() uint64 {
	var a, b uint64 = 0, 1

	return func() uint64 {
		defer func() {
			a, b = b, a+b
		}()
		return a
	}
}

type UrlLog struct {
	UrlFilePath      string
	LogFilePath      string
	UrlChan          chan string
	HttpResponseChan chan HttpResponse
	WG               *sync.WaitGroup
	Lock             sync.RWMutex
}

type HttpResponse struct {
	Url        string
	Response   string
	StatusCode int
	Time       time.Time
}

var cnt atomic.Int32

// Process the url file and write the log of url in log file
func (u *UrlLog) ReadFile() {
	defer u.WG.Done()
	file, err := os.Open(u.UrlFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	defer close(u.UrlChan)
	for scanner.Scan() {
		u.UrlChan <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

func (u *UrlLog) RequestUrl(url string) {
	defer u.WG.Done()
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	// body, err := ioutil.ReadAll(resp.Body)
	u.HttpResponseChan <- HttpResponse{
		Url: url,
		// Response:   string(body),
		StatusCode: resp.StatusCode,
		Time:       time.Now(),
	}
}

func (u *UrlLog) WriteFile(file string) {
	defer u.WG.Done()
	u.Lock.RLock()
	defer u.Lock.RUnlock()
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	for resp := range u.HttpResponseChan {
		cnt.Add(-1)
		fmt.Println(resp)
		bytes, err := json.Marshal(resp)
		if err != nil {
			fmt.Println(err)
		}
		// err = ioutil.WriteFile(file, bytes, 0644)
		// if err != nil {
		// 	fmt.Println(err)
		// }
		if _, err := f.WriteString(string(bytes) + "\n"); err != nil {
			log.Println(err)
		}
	}
}

func StartUrlLog() {
	var urlLog UrlLog
	urlLog.UrlFilePath = "url.txt"
	urlLog.LogFilePath = "url_log.txt"
	urlLog.WG = &sync.WaitGroup{}
	urlLog.UrlChan = make(chan string)
	urlLog.HttpResponseChan = make(chan HttpResponse)
	urlLog.WG.Add(3)
	go urlLog.ReadFile()
	go func() {
		defer urlLog.WG.Done()
		for {
			if url, ok := <-urlLog.UrlChan; ok {
				urlLog.WG.Add(1)
				cnt.Add(1)
				go urlLog.RequestUrl(url)
				continue
			}
			break
		}
		for {
			if cnt.Load() == 0 {
				close(urlLog.HttpResponseChan)
				break
			}
		}
	}()
	go urlLog.WriteFile(urlLog.LogFilePath)
	urlLog.WG.Wait()
	// var ch chan struct{} = make(chan struct{})
	// <-ch
}
