// Copyright © 2018 Kasisnu <kasisnu.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/google/go-github/github"
	git "github.com/libgit2/git2go"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/oauth2"
)

var cfgFile string
var concurrency int

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-recurse",
	Short: "Download every git repo under a github organisation - concurrently",
	Long: `gh-recurse downloads every git repository under a github organisation

	ex. GITHUB_OAUTH_TOKEN=your-fancy-token gh-recurse github

	will download every github repository under the github organisation
	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Fatal("missing organisation name. see --help")
	}
	orgName := args[0]

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

	opt = &github.RepositoryListByOrgOptions{Type: "forks"}
	forks, _, err := client.Repositories.ListByOrg(ctx, orgName, opt)
	if err != nil {
		log.Fatal(err)
	}

	repos = append(repos, forks...)

	cloneOptions := &git.CloneOptions{}
	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback:      getPassphraseCredentialsCallback(),
			CertificateCheckCallback: certificateCheckCallback,
		},
	}

	var wg sync.WaitGroup
	wg.Add(len(repos))
	repoCh := make(chan *github.Repository)

	for i := 0; i < concurrency; i++ {
		go func() {
			for repo := range repoCh {
				defer wg.Done()
				log.Printf("Working on: %s", *repo.Name)
				_, err := git.Clone(fmt.Sprintf("git@github.com:%s/%s", orgName, *repo.Name), *repo.Name, cloneOptions)
				if err != nil {
					if strings.Contains(err.Error(), "exists and is not an empty directory") {
						log.Println(err)
						continue
					}

					log.Fatal(err)
				}
			}
		}()
	}

	for _, repo := range repos {
		repoCh <- repo
	}
	close(repoCh)

	wg.Wait()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gh-recurse.yaml)")
	rootCmd.PersistentFlags().IntVar(&concurrency, "concurrency", 4, "")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gh-recurse" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gh-recurse")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
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
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		ret, cred := git.NewCredSshKey(
			"git",
			filepath.Join(home, ".ssh/id_rsa.pub"),
			filepath.Join(home, ".ssh/id_rsa"),
			passphrase,
		)
		return git.ErrorCode(ret), &cred
	}
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	return 0
}
