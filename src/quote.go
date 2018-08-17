package main

import(
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"encoding/json"
)

type quoteResponse struct {
	Quotes []quote `json:"quotes"`
}

type quote struct {
	Author string `json:"author"`
	Quote string `json:"quote"`
	Tags []string `json:"tags"`
}

func main() {
	http.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		resp, err := http.Get("http://opinionated-quotes-api.gigalixirapp.com/v1/quotes")
		if err != nil {
			fmt.Printf("Error fetching quote: %v\n", err)
			return
		}
		rawQuote, err := ioutil.ReadAll(resp.Body)
		var quoteResp quoteResponse
		err = json.Unmarshal(rawQuote, &quoteResp)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			return
		}
		quoteJSON, err := json.Marshal(quoteResp.Quotes[0])
		if err != nil {
			fmt.Printf("Error converting to JSON: %v\n", err)
			return
		}
		if err != nil {
			fmt.Printf("Error reading quote from response: %v\n", err)
			return
		}
		fmt.Println(string(quoteJSON))
		io.WriteString(w, string(quoteJSON))
	})
	fmt.Println("Server Started...")

	http.ListenAndServe(":6213", nil)
}
