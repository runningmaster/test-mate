package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var (
	sections = [...]string{
		"kupit-mate-2.html",
		"paragvayskiy-mate.html",
		"brazilskiy-mate.html",
	}
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	start := time.Now()

	wg := new(sync.WaitGroup)
	ch := make(chan string)

	wg.Add(len(sections))
	for _, v := range sections {
		go grabSection(makeCompleteURL(v), wg, ch)
	}

	results := make([]string, 0, 100)
	go func(wg *sync.WaitGroup) {
		for v := range ch {
			if strings.HasPrefix(v, "+") {
				results = append(results, strings.Replace(v, "+", "", 1))
			} else {
				wg.Add(1)
				go grabPage(v, wg, ch)
			}
		}
	}(wg)

	wg.Wait()
	close(ch)

	// It is filter output for poors :)
	var filterText string
	if len(os.Args) > 1 {
		filterText = os.Args[1]
	}

	var count int
	for _, v := range results {
		if strings.Contains(strings.ToLower(v), strings.ToLower(filterText)) {
			fmt.Println(v)
			count++
		}
	}

	fmt.Printf("Count: %d Elapsed time: %s\n", count, time.Since(start).String())
}

func grabSection(url string, wg *sync.WaitGroup, c chan<- string) {
	defer wg.Done()

	d, err := goquery.NewDocument(url)
	panicIfError(err)

	d.Find("#content table a").Each(func(i int, s *goquery.Selection) {
		if l, ok := s.Attr("href"); ok && strings.HasPrefix(l, "http") {
			c <- l
		}
	})
}

func grabPage(url string, wg *sync.WaitGroup, c chan<- string) {
	defer wg.Done()

	b := new(bytes.Buffer)
	r, err := http.Get(url)
	panicIfError(err)

	defer r.Body.Close()
	_, err = io.Copy(b, r.Body)
	panicIfError(err)

	d, err := goquery.NewDocumentFromReader(convRdrWin1251toUTF8(b))
	panicIfError(err)
	d.Find("div h3 a").Each(func(i int, s *goquery.Selection) {
		price := ""
		s.Parent().Parent().Find("span").Each(func(i int, s *goquery.Selection) {
			price = strings.TrimSpace(s.Text())
		})
		exists := false
		s.Parent().Parent().Find(".inputbox").Each(func(i int, s *goquery.Selection) {
			exists = true
		})
		if exists {
			// The prefix "+" here - it is small workaround
			c <- "+" + strings.TrimSpace(s.Text()) + " " + price
		}
	})
}

func makeCompleteURL(url string) string {
	return fmt.Sprintf("http://mate-kiev.com/shop/%s", url)
}

func convRdrWin1251toUTF8(r io.Reader) io.Reader {
	return transform.NewReader(r, charmap.Windows1251.NewDecoder())
}

func convStrWin1251toUTF8(s string) string {
	b := new(bytes.Buffer)
	r := transform.NewReader(strings.NewReader(s), charmap.Windows1251.NewEncoder())
	_, err := io.Copy(b, r)
	panicIfError(err)
	return b.String()
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
