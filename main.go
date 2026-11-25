package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/google/go-github/v42/github"
	"github.com/gorilla/mux"
	engine "github.com/reviewpad/reviewpad"
	"github.com/reviewpad/reviewpad/collector"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type DryRunRequest struct {
	GitHubToken    *string `json:"gitHubToken"`
	PullRequestUrl *string `json:"pullRequestUrl"`
	Configuration  *string `json:"configuration"`
}

func dryRun(w http.ResponseWriter, r *http.Request) {
	dryRun := DryRunRequest{}

	err := json.NewDecoder(r.Body).Decode(&dryRun)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error decoding response object", http.StatusBadRequest)
		return
	}

	if dryRun.Configuration == nil || *dryRun.Configuration == "" {
		http.Error(w, "configuration is required", http.StatusBadRequest)
		return
	}

	// The configuration is checked first so syntax errors need no PR or token.
	buf := bytes.NewBuffer([]byte(*dryRun.Configuration))
	file, err := engine.Load(buf)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error loading configuration: "+err.Error(), http.StatusBadRequest)
		return
	}

	if dryRun.PullRequestUrl == nil || dryRun.GitHubToken == nil || *dryRun.PullRequestUrl == "" || *dryRun.GitHubToken == "" {
		http.Error(w, "Configuration is valid. Provide pullRequestUrl and gitHubToken to run it.", http.StatusBadRequest)
		return
	}

	pullRequestDetailsRegex := regexp.MustCompile(`github\.com\/([^/]+)\/([^/]+)\/pull\/(\d+)`)
	pullRequestDetails := pullRequestDetailsRegex.FindSubmatch([]byte(*dryRun.PullRequestUrl))
	if pullRequestDetails == nil {
		http.Error(w, "Invalid pull request URL", http.StatusBadRequest)
		return
	}

	repositoryOwner := string(pullRequestDetails[1][:])
	repositoryName := string(pullRequestDetails[2][:])
	pullRequestNumber, err := strconv.Atoi(string(pullRequestDetails[3][:]))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error converting pull request number", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *dryRun.GitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	gitHubClient := github.NewClient(tc)
	gitHubClientGQL := githubv4.NewClient(tc)
	collectorClient := collector.NewCollector("", repositoryOwner)

	ghPullRequest, _, err := gitHubClient.PullRequests.Get(ctx, repositoryOwner, repositoryName, pullRequestNumber)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error getting pull request: "+err.Error(), http.StatusBadRequest)
		return
	}

	program, err := engine.Run(ctx, gitHubClient, gitHubClientGQL, collectorClient, ghPullRequest, file, true)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error running configuration: "+err.Error(), http.StatusBadRequest)
		return
	}

	response, err := json.Marshal(program)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error marshal program", http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}

// withCORS lets the web UI call the API from another origin.
// Override the allowed origin with ALLOWED_ORIGIN.
func withCORS(next http.Handler) http.Handler {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/dry-run", dryRun).Methods("POST", "OPTIONS")
	log.Fatal(http.ListenAndServe(":8080", withCORS(router)))
}
