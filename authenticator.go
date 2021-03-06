package main

import (
	"encoding/json"
	"github.com/howeyc/gopass"
	"io"
	"io/ioutil"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"time"
)

var (
  email = kingpin.Flag("email", "Your Apple ID (requires CasperJS)").PlaceHolder("sjobs@apple.com").String()
  teamId = kingpin.Flag("team", "Your Apple Developer Team ID (requires CasperJS)").String()
  askPassword = kingpin.Flag("password", "Ask for AppleID password (requires CasperJS)").Short('p').Bool()
  password string
)

type Authenticator struct {
	cookies       []*http.Cookie
	authenticated bool
}

func NewAuthenticator() *Authenticator {
	if *askPassword {
		log.Print("Password: ")
		password = string(gopass.GetPasswd())
		*askPassword = false
	}

	return &Authenticator{
		cookies:       make([]*http.Cookie, 0),
		authenticated: false,
	}
}

func (a *Authenticator) IsAuthenticated() bool {
	return a.authenticated
}

func (a *Authenticator) Authenticate() (err error) {
	if a.authenticated {
		return
	}

	a.authenticated = true

	err = a.loadCookiesViaCasper()
	return
}

func (a *Authenticator) loadCookiesViaCasper() (err error) {
	casper, err := exec.LookPath("casperjs")

	if err != nil {
		log.Println("To authenticate against Apple `wwdcdl` requires the headless browser CasperJS (http://casperjs.org).")

		if runtime.GOOS == "darwin" {
			log.Println("If you have Homebrew installed, just use `brew install casperjs`.\n")
		}

		log.Fatalln("CasperJS not installed.")
	}

	dir, err := ioutil.TempDir("", "wwdcdl")

	if err != nil {
		return
	}

	fileName := path.Join(dir, "cookies.json")

	scriptFileName := path.Join(dir, "login.coffee")
	asset, _ := Asset("data/login.coffee")
	ioutil.WriteFile(scriptFileName, asset, 0600)

	cmd := exec.Command(casper, scriptFileName, fileName, *email, password, *teamId)

	stdout, _ := cmd.StdoutPipe()
	go io.Copy(os.Stdout, stdout)

	stderr, _ := cmd.StderrPipe()
	go io.Copy(os.Stderr, stderr)

	stdin, err := cmd.StdinPipe()
	go io.Copy(stdin, os.Stdin)

	err = cmd.Run()

	if err != nil {
		return
	}

	a.loadCookiesFromFile(fileName)

	return
}

func (a *Authenticator) loadCookiesFromFile(fileName string) (err error) {
	data, err := ioutil.ReadFile(fileName)
	a.loadCookies(data)
	return
}

func (a *Authenticator) loadCookies(data []byte) {
	var rawCookies []map[string]interface{}

	json.Unmarshal(data, &rawCookies)

	for _, rawCookie := range rawCookies {
		cookie := &http.Cookie{
			Name:       rawCookie["name"].(string),
			Value:      rawCookie["value"].(string),
			Path:       rawCookie["path"].(string),
			Domain:     rawCookie["domain"].(string),
			Expires:    time.Unix(int64(rawCookie["expiry"].(float64)), 0),
			RawExpires: rawCookie["expires"].(string),
			MaxAge:     0,
			Secure:     rawCookie["secure"].(bool),
			HttpOnly:   rawCookie["httponly"].(bool),
		}

		a.cookies = append(a.cookies, cookie)
	}

	log.Printf("Imported %d cookies.\n", len(a.cookies))

	cookieUrl, _ := url.Parse("https://apple.com")
	http.DefaultClient.Jar, _ = cookiejar.New(nil)
	http.DefaultClient.Jar.SetCookies(cookieUrl, a.cookies)
}

func (a *Authenticator) Extend(req *http.Request) (err error) {
	err = a.Authenticate()
	return
}
