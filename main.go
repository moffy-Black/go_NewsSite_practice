package main

import (
	"bytes"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"work/news"
	"github.com/joho/godotenv"
)

var tpl = template.Must(template.ParseFiles("index.html")) //提供されたファイルからのテンプレート定義を指すパッケージレベルの変数です。

type Search struct {
	Query      string
	NextPage   int
	TotalPages int
	Results    *news.Results
}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}

	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// tpl.Execute(w, nil)
	// w.Write([]byte("<h1>Hello World!</h1>")) /*w:httpリクエスト r:クライアントから受信したhttpリクエスト,POST*/
	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}

func searchHandler(newsapi *news.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(r.URL.String())
		if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
		}


		params := u.Query()
		searchQuery := params.Get("q")
		page := params.Get("page")
		if page == "" {
			page = "1"
		}

		results, err := newsapi.FetchEverything(searchQuery, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nextPage, err := strconv.Atoi(page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		search := &Search{
			Query:      searchQuery,
			NextPage:   nextPage,
			TotalPages: int(math.Ceil(float64(results.TotalResults) / float64(newsapi.PageSize))),
			Results:    results,
		}

		if ok := !search.IsLastPage(); ok {
			search.NextPage++
		}

		buf := &bytes.Buffer{}
		err = tpl.Execute(buf, search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		buf.WriteTo(w)
	}
}

func main() {
	err := godotenv.Load() /*Load()は.envファイルを読み取り、設定された変数を環境にload*/
	if err != nil { /*関数タイプの変数がゼロの値であるとき*/
		log.Println("Error loading .env file")
	} /*環境に秘密の資格情報を保存するときに役立つ*/

	port := os.Getenv("PORT") /*環境変数portの値をとってくる*/
	if port == "" { /*もし、とってこれなかったら*/
		port = "3000"
	}

	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
	}

	myClient := &http.Client{Timeout: 10 * time.Second}
	newsapi := news.NewClient(myClient, apiKey, 20)

	fs := http.FileServer(http.Dir("assets")) //べての静的ファイルが配置されているディレクトリを渡すことによって、ファイルサーバーオブジェクトをインスタンス化することです。

	mux := http.NewServeMux() /*URIとhtmlを照合するマルチプレクサ(handle関数を呼び出す)*/
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs)) ///assets/プレフィックスで始まるすべてのパスにこのファイルサーバーオブジェクトを使用するようにルーターに指示する
	mux.HandleFunc("/search", searchHandler(newsapi))
	mux.HandleFunc("/", indexHandler)
	http.ListenAndServe(":"+port, mux)
}