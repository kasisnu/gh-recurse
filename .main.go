package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/google/go-github/github"
	git "github.com/libgit2/git2go"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/oauth2"
)

func main() {
	orgName := os.Args[1]

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_OAUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := client.Repositories.ListByOrg(ctx, orgName, opt)
	if err != nil {
		log.Fatal(err)
	}

	cloneOptions := &git.CloneOptions{}
	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback:      getPassphraseCredentialsCallback(),
			CertificateCheckCallback: certificateCheckCallback,
		},
	}

	for _, repo := range repos {
		log.Println("Working on: ", *repo.Name)
		_, err := git.Clone(fmt.Sprintf("git@github.com:%s/%s", orgName, *repo.Name), *repo.Name, cloneOptions)
		if err != nil {
			if strings.Contains(err.Error(), "exists and is not an empty directory") {
				log.Println(err)
				continue
			}

			log.Fatal(err)
		}
	}
}

func getPassphraseCredentialsCallback() func(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	var passphrase = ""
	fmt.Println("Enter passphrase: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	passphrase = string(bytePassword)

	return func(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
		ret, cred := git.NewCredSshKey("git", "/Users/kasisnu/.ssh/id_rsa.pub", "/Users/kasisnu/.ssh/id_rsa", passphrase)
		return git.ErrorCode(ret), &cred
	}
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	return 0
}
