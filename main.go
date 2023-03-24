package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pmezard/go-difflib/difflib"
)

type Watcher struct {
	Url       string
	Css       string
	Html      string
	IsFetched bool
	UpdatedAt time.Time
}

type Email struct {
	Subject string
	Content string
}

func main() {
	config.load("config.json")
	var (
		smtpConfig  = config.SMTP
		watchConfig = config.Watch
		user        = smtpConfig.User
		pass        = smtpConfig.Pass
		host        = smtpConfig.Host
		port        = smtpConfig.Port
		from        = smtpConfig.From
		to          = smtpConfig.To
	)

	watchlist := []Watcher{}
	for _, watch := range watchConfig {
		watcher := Watcher{
			Url:       watch.Url,
			Css:       watch.Css,
			Html:      "",
			IsFetched: false,
		}
		if watcher.Css == "" {
			watcher.Css = "body"
		}
		watchlist = append(watchlist, watcher)
	}
	resultChannel := make(chan Email)
	quit := make(chan bool)
	for _, watcher := range watchlist {
		go func(watcher Watcher) {
			err := watcher.watch(resultChannel)
			if err != nil {
				log.Fatal(err)
				quit <- true
			}
		}(watcher)
	}
	for {
		select {
		case email := <-resultChannel:
			log.Printf("%s\r\n", email.Subject)
			auth := smtp.PlainAuth("", user, pass, host)
			message := fmt.Sprintf(
				"From: %s\r\nSubject: %s\r\n\r\n%s", from, email.Subject, email.Content)
			err := smtp.SendMail(host+":"+port, auth, from, to, []byte(message))
			if err != nil {
				log.Fatal(err)
				return
			}
		case <-quit:
			log.Println("quit")
			return
		}
	}
}

func (w *Watcher) watch(result chan Email) (err error) {
	for {
		html, err := w.fetch()
		if err != nil {
			return err
		}
		if w.Html != html {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(w.Html),
				B:        difflib.SplitLines(html),
				FromFile: "Before",
				ToFile:   "After",
			}
			diffrence, _ := difflib.GetUnifiedDiffString(diff)
			w.Html = html
			w.UpdatedAt = time.Now()
			if w.IsFetched {
				result <- Email{
					Subject: fmt.Sprintf("[%s] %s 변화", w.UpdatedAt.Format("15:04:05"), w.Url),
					Content: fmt.Sprintf("아래와 같은 변화가 감지되었습니다.\n\n%s", diffrence),
				}
			} else {
				log.Printf("[등록] %s\r\n", w.Url)
				w.IsFetched = true
			}
		}
		time.Sleep(time.Duration(config.DelayTime) * time.Minute)
	}
}

func (w *Watcher) fetch() (html string, err error) {
	res, err := http.Get(w.Url)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = errors.New(
			fmt.Sprintf("status code error: %d %s", res.StatusCode, res.Status))
		return
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}
	node := doc.Find(w.Css)
	html, err = node.Html()
	return
}