# gh-recurse


This does nothing too useful and doesn't do anything you couldn't do with some bash-foo. It downloads all repositories under a github organisation with one command.

It was a fun way to revise some Golang and pass time while waiting for a long download to finish.

Have fun on your first day at work.

## Installation
```
go get -u github.com/kasisnu/gh-recurse

```

ex,

  ```
  gh-recurse google # Download all repositories under the Google Github organisation

  gh-recurse google --type forks # Download only things that Google forked

  gh-recurse google --type private # Download only repositories internal to Google(requires privileged access token)

  gh-recurse google --type private --dry-run # Show me what you would download

  gh-recurse --help
  ```
