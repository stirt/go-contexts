package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

func GetWithTimeout(cxt context.Context, url string, t time.Duration) (*http.Response, error) {
	out := make(chan *http.Response)
	errchan := make(chan error)
	cxt, cancel := context.WithTimeout(cxt, t)
	defer cancel()
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			errchan <- err
			return
		}
		out <- resp
	}()
	for {
		select {
		case <-cxt.Done():
			return &http.Response{}, cxt.Err()
		case resp := <-out:
			return resp, nil
		case err := <-errchan:
			return &http.Response{}, err
		}
	}
}

func getTodo(cxt context.Context, id int) ([]byte, error) {
	var b []byte
	url := fmt.Sprintf("https://jsonplaceholder.typicode.com/photos/%d", id)
	resp, err := GetWithTimeout(cxt, url, time.Second*5)
	if err != nil {
		return b, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getTodos(cxt context.Context) (chan []byte, chan error) {
	bodychan := make(chan []byte)
	errchan := make(chan error)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b, err := getTodo(cxt, i)
			if err != nil {
				errchan <- err
				return
			}
			bodychan <- b
		}(i)
	}
	go func() {
		wg.Wait()
		close(bodychan)
	}()
	return bodychan, errchan
}

func main() {
	cxt := context.Background()
	bodychan, errchan := getTodos(cxt)
	for {
		select {
		case b, ok := <-bodychan:
			if !ok {
				close(errchan)
				return
			}
			fmt.Println(string(b))
		case err := <-errchan:
			fmt.Println(err)
		}
	}
}
