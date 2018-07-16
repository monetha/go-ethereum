# go-ethereum [![GoDoc][1]][2] [![Build Status][3]][4] [![Go Report Card][5]][6] [![Coverage Status][7]][8]

[1]: https://godoc.org/github.com/monetha/go-ethereum?status.svg
[2]: https://godoc.org/github.com/monetha/go-ethereum
[3]: https://travis-ci.org/monetha/go-ethereum.svg?branch=master
[4]: https://travis-ci.org/monetha/go-ethereum
[5]: https://goreportcard.com/badge/github.com/monetha/go-ethereum
[6]: https://goreportcard.com/report/github.com/monetha/go-ethereum
[7]: https://codecov.io/gh/monetha/go-ethereum/branch/master/graph/badge.svg
[8]: https://codecov.io/gh/monetha/go-ethereum

Packages to simplify work with the Ethereum blockchain

## Contributing

If you'd like to add new exported APIs, please [open an issue][open-issue]
describing your proposal &mdash; discussing API changes ahead of time makes
pull request review much smoother. In your issue, pull request, and any other
communications, please remember to treat your fellow contributors with
respect!

### Setup

[Fork][fork], then clone the repository:

```
mkdir -p $GOPATH/src/github.com/monetha
cd $GOPATH/src/github.com/monetha
git clone git@github.com:your_github_username/go-ethereum.git
cd go-ethereum
git remote add upstream https://github.com/monetha/go-ethereum.git
git fetch upstream
```

Install dependencies:

```
make dependencies
```

Make sure that the tests and the linters pass:

```
make test
make lint
```

### Making Changes

Start by creating a new branch for your changes:

```
cd $GOPATH/src/github.com/monetha/go-ethereum
git checkout master
git fetch upstream
git rebase upstream/master
git checkout -b cool_new_feature
```

Make your changes, then ensure that `make lint` and `make test` still pass. If
you're satisfied with your changes, push them to your fork.

```
git push origin cool_new_feature
```

Then use the GitHub UI to open a pull request.

At this point, you're waiting on us to review your changes. We *try* to respond
to issues and pull requests within a few business days, and we may suggest some
improvements or alternatives. Once your changes are approved, one of the
project maintainers will merge them.

We're much more likely to approve your changes if you:

* Add tests for new functionality.
* Write a [good commit message][commit-message].
* Maintain backward compatibility.

[fork]: https://github.com/monetha/go-ethereum/fork
[open-issue]: https://github.com/monetha/go-ethereum/issues/new
[commit-message]: http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html