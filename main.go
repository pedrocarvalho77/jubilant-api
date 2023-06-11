package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/google/go-github/v42/github"
	"github.com/gorilla/mux"
	"github.com/reviewpad/reviewpad"
	"github.com/reviewpad/reviewpad/collector"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type DryRunRequest struct {
	GitHubToken            *string `json:"gitHubToken"`
	PullRequestUrl         *string `json:"pullRequestUrl"`
	ReviewpadConfiguration *string `json:"reviewpadConfiguration"`
}

func dryRun(w http.ResponseWriter, r *http.Request) {
	dryRun := DryRunRequest{}

	err := json.NewDecoder(r.Body).Decode(&dryRun)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error decoding response object", http.StatusBadRequest)
		return
	}

	pullRequestDetailsRegex := regexp.MustCompile(`github\.com\/(.+)\/(.+)\/pull\/(\d+)`)
	pullRequestDetails := pullRequestDetailsRegex.FindSubmatch([]byte(*dryRun.PullRequestUrl))

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
		http.Error(w, "Error getting pull request", http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer([]byte(*dryRun.ReviewpadConfiguration))
	file, err := reviewpad.Load(buf)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error loading reviewpad configuration", http.StatusBadRequest)
		return
	}

	program, err := reviewpad.Run(ctx, gitHubClient, gitHubClientGQL, collectorClient, ghPullRequest, file, true)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error running reviewpad", http.StatusBadRequest)
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

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/dry-run", dryRun).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}
