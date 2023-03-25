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
	var watchConfig = config.Watch

	watchlist := []*Watcher{}
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
		watchlist = append(watchlist, &watcher)
	}
	err := async_routine(watchlist)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func routine(watchlist []*Watcher) (err error) {
	/**
	 * 요청 과다를 막기 위한 sync 방법 (monkeypatch)
	 */
	var (
		smtpConfig = config.SMTP
		user       = smtpConfig.User
		pass       = smtpConfig.Pass
		host       = smtpConfig.Host
		port       = smtpConfig.Port
		from       = smtpConfig.From
		to         = smtpConfig.To
	)
	for {
		var changedList = []*Watcher{}
		for _, w := range watchlist {
			html, err := w.fetch()
			if err != nil {
				return err
			}
			if w.Html != html {
				w.Html = html
				w.UpdatedAt = time.Now()
				if w.IsFetched {
					changedList = append(changedList, w)
				} else {
					log.Printf("[등록] %s\r\n", w.Url)
					w.IsFetched = true
				}
			}
		}
		if len(changedList) > 0 {
			email := Email{
				Subject: fmt.Sprintf("[%s] %s 등 %d개",
					changedList[0].UpdatedAt.Format("15:04:05"), changedList[0].Url, len(changedList)),
				Content: "감지된 사이트 목록\r\n",
			}
			for _, changed := range changedList {
				email.Content += changed.UpdatedAt.Format("15:04:05") + " - " + changed.Url + "\r\n"
			}
			log.Printf("%s\r\n", email.Subject)
			auth := smtp.PlainAuth("", user, pass, host)
			body := fmt.Sprintf("From: %s\r\nSubject: %s\r\n\r\n%s", from, email.Subject, email.Content)
			err = smtp.SendMail(host+":"+port, auth, from, to, []byte(body))
			if err != nil {
				return err
			}
		}
		time.Sleep(time.Duration(config.DelayTime) * time.Minute)
	}
}

func async_routine(watchlist []*Watcher) (err error) {
	var (
		smtpConfig = config.SMTP
		user       = smtpConfig.User
		pass       = smtpConfig.Pass
		host       = smtpConfig.Host
		port       = smtpConfig.Port
		from       = smtpConfig.From
		to         = smtpConfig.To
	)
	resultChannel := make(chan Email)
	quit := make(chan bool)
	for _, watcher := range watchlist {
		go func(watcher *Watcher) {
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
			body := fmt.Sprintf("From: %s\r\nSubject: %s\r\n\r\n%s", from, email.Subject, email.Content)
			err := smtp.SendMail(host+":"+port, auth, from, to, []byte(body))
			if err != nil {
				log.Fatal(err)
				return err
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
					Content: fmt.Sprintf("%s", diffrence),
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
