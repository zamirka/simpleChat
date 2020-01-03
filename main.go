package main

import (
	"chat/trace"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/stretchr/objx"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
)

// templ represents a single template
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServerHttp handles the HTTP request
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

// Secrets is a structure for storing secrets
// file structure
// {
//     "secrets":[
//         {
//             "secretName":"chat",
//             "key":"",
//             "secret":"",
//             "url":""
//         }
//     ]
// }
type Secrets struct {
	Secrets []Secret `json:"secrets"`
}

// Secret is a container for storing one secret
type Secret struct {
	SecretName string `json:"secretName"`
	Key        string `json:"key"`
	Secret     string `json:"secret"`
	URL        string `json:"url"`
}

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse() // Parse the flags
	// read secrets from a file
	secrets, err := readSecretsFromJSONFile("secrets.json")
	if err != nil {
		log.Fatal("readSecretsFromJSONFile(\"secrets.json\")", err)
	}
	secretsMap := make(map[string]Secret)
	for _, s := range secrets.Secrets {
		secretsMap[s.SecretName] = s
	}
	// setup gominauth
	gomniauth.SetSecurityKey(secretsMap["chat"].Secret)
	gomniauth.WithProviders(
		facebook.New(secretsMap["facebook"].Key, secretsMap["facebook"].Secret, secretsMap["facebook"].URL),
		github.New(secretsMap["github"].Key, secretsMap["github"].Secret, secretsMap["github"].URL),
		google.New(secretsMap["google"].Key, secretsMap["google"].Secret, secretsMap["google"].URL),
	)
	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	// root
	http.Handle("/", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.HandleFunc("/logout/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/room", r)
	// get the room going
	go r.run()
	// strart the web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func readSecretsFromJSONFile(filename string) (*Secrets, error) {
	jsonFile, err := os.Open(filename)
	defer jsonFile.Close()
	if err != nil {
		return nil, err
	}
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	var secrets Secrets

	err = json.Unmarshal(byteValue, &secrets)
	if err != nil {
		return nil, err
	}
	return &secrets, nil
}
