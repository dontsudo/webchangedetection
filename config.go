package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

/** 네이버 이메일 사용시
{
	"smtp": {
		"user": "네이버아이디",
		"pass": "네이버비밀번호",
		"host": "smtp.naver.com",
		"port": "587",
		"from": "네이버이메일",
		"to": ["보낼이메일1", "보낼이메일2"]
	},
	"watch": [
		{
			"url": "https://www.naver.com",
		}
	]
}
*/

type Config struct {
	SMTP      SMTPConfig    `json:"메일"`
	Watch     []WatchConfig `json:"사이트"`
	DelayTime float64       `json:"딜레이"`
}

type SMTPConfig struct {
	User string   `json:"아이디"`
	Pass string   `json:"비밀번호"`
	Host string   `json:"서버명"`
	Port string   `json:"포트정보"`
	From string   `json:"발신인"`
	To   []string `json:"수신인"`
}

type WatchConfig struct {
	Url string `json:"주소"`
	Css string `json:"선택자"`
}

var config Config

func (c *Config) load(filename string) (err error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return
	}
	log.Print("[설정] OK\n")

	return nil
}
