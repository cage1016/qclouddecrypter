package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/go-querystring/query"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/tylerb/graceful.v1"
	"gopkg.in/yaml.v2"

	"github.com/cage1016/qclouddecrypter/crypto"
)

const (
	empty      = ""
	tab        = "\t"
	yamlPath   = ".qclouddecrypter/secretkey.yaml"
	urlPattern = `https://%s/oauth2/connect?%s&cb=http://127.0.0.1:%d`
)

var (
	answers Answers
	green   = color.New(color.FgHiGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
)

type App struct {
	Name     string
	Id       string
	Secret   string
	Provider string
	Scopes   []string
}

type Yaml struct {
	Apps []App
}

type Answers struct {
	Server   string
	Name     string `json:",omitempty"`
	Id       string
	Secret   string
	Provider string
	Scopes   string
	Port     int
}

type Options struct {
	Id       string `url:"app_id"`
	Provider string `url:"provider"`
	Scopes   string `url:"scope"`
}

func prettyJSON(data interface{}) (string, error) {
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent(empty, tab)

	err := encoder.Encode(data)
	if err != nil {
		return empty, err
	}
	return buffer.String(), nil
}

func getYaml() (data []byte, err error) {
	var u *user.User
	u, err = user.Current()
	if err != nil {
		return
	}

	var f *os.File
	f, err = os.Open(filepath.Join(u.HomeDir, yamlPath))
	if err != nil {
		return
	}
	defer f.Close()

	data, err = ioutil.ReadAll(f)
	if err != nil {
		return
	}
	return
}

func getServerQuestion() *survey.Question {
	return &survey.Question{
		Name: "server",
		Prompt: &survey.Select{
			Message: "Choose Auth Server:",
			Options: []string{"connector.myqnapcloud.com", "connector.alpha-myqnapcloud.com"},
			Default: "connector.alpha-myqnapcloud.com",
		},
		Validate: survey.Required,
	}
}

func getSecretQuestion() (*Yaml, error) {
	data, err := getYaml()
	if err != nil {
		return nil, err
	}

	y := Yaml{}
	err = yaml.Unmarshal(data, &y)
	if err != nil {
		return nil, err
	}
	return &y, nil
}

func getAnswers() (*Answers, error) {
	var qs = []*survey.Question{
		getServerQuestion(),
	}

	yaml, err := getSecretQuestion()
	if err != nil {
		fmt.Printf("\n%s\n\n", red(err.Error(), ", manual input"))

		q2 := &survey.Question{
			Name: "id",
			Prompt: &survey.Input{
				Message: "Input App Id:",
				Default: "",
			},
			Validate: survey.Required,
		}

		q3 := &survey.Question{
			Name: "secret",
			Prompt: &survey.Input{
				Message: "Input App Secret:",
				Default: "",
			},
			Validate: survey.Required,
		}

		q4 := &survey.Question{
			Name: "provider",
			Prompt: &survey.Input{
				Message: "Input auth provider:",
				Default: "",
			},
			Validate: survey.Required,
		}

		q5 := &survey.Question{
			Name: "scopes",
			Prompt: &survey.Input{
				Message: "Input Scope (separated by space):",
				Default: "",
			},
		}

		q6 := &survey.Question{
			Name: "port",
			Prompt: &survey.Input{
				Message: "Port to run app on:",
				Default: "3000",
			},
			Validate: survey.Required,
		}

		qs = append(qs, q2, q3, q4, q5, q6)

		err = survey.Ask(qs, &answers)
		if err != nil {
			return nil, err
		}
	} else {
		q2 := &survey.Question{
			Name: "name",
			Prompt: &survey.Select{
				Message: "Choose App:",
				Options: func() (o []string) {
					for _, v := range yaml.Apps {
						o = append(o, v.Name)
					}
					return
				}(),
			},
			Validate: survey.Required,
		}

		q3 := &survey.Question{
			Name: "port",
			Prompt: &survey.Input{
				Message: "Port to run app on:",
				Default: "3000",
			},
			Validate: survey.Required,
		}

		qs = append(qs, q2, q3)

		err = survey.Ask(qs, &answers)
		if err != nil {
			return nil, err
		}

		for _, v := range yaml.Apps {
			if v.Name == answers.Name {
				answers.Id = v.Id
				answers.Secret = v.Secret
				answers.Provider = v.Provider
				answers.Scopes = strings.Join(v.Scopes, " ")
				break
			}
		}
	}

	return &answers, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	m, err := url.ParseQuery(r.URL.String())
	if err != nil {
		fmt.Fprintf(w, "%s", err)
	}

	cryptoService, _ := crypto.NewCryptoService(answers.Secret)
	cret, err := cryptoService.DecryptCredentials(m.Get("/?result"))
	if err != nil {
		fmt.Fprintf(w, "parser error => %v", err)
	}

	s, _ := prettyJSON(cret)
	fmt.Fprintf(w, "%s", s)
}

func main() {
	a, err := getAnswers()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	s, _ := prettyJSON(a)
	fmt.Printf("%s", s)

	v, _ := query.Values(Options{
		Id:       a.Id,
		Provider: a.Provider,
		Scopes:   a.Scopes,
	})

	fmt.Println(green("!"), "Visit", green(fmt.Sprintf(urlPattern, a.Server, v.Encode(), a.Port)))

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	graceful.Run(fmt.Sprintf(":%d", a.Port), 10*time.Second, mux)
}
