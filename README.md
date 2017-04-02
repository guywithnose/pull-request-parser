pull-request-parser uses the Github API to parse open pull requests and aggregate useful information about them. This includes information like whether the pull request is rebased, how many people have approved it, and the status of any CI builds.

### Setup
```sh
go get github.com/guywithnose/pull-request-parser/prp
prp --config ~/prpConfig.json init-config
prp --config ~/prpConfig.json profile add default --token {YOUR_GITHUB_TOKEN}
prp --config ~/prpConfig.json repo add {USER} {REPO_NAME}
```

#### Parse
```sh
prp --config ~/prpConfig.json parse
```
Parses pull requests on tracked repositories and outputs to the screen

#### Auto-Rebase
```sh
prp --config ~/prpConfig.json repo set-path {USER}/{REPO_NAME} {PATH_TO_LOCAL_CLONE}
prp --config ~/prpConfig.json auto-rebase
```
Parses your pull requests on tracked repositories and if they are not rebased it will try to update them.  This is especially useful for git workflows that only allow fast-forwards.
