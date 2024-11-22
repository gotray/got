# got

A tool to run Go with Python environment configured.

[![Build Status](https://github.com/gotray/got/actions/workflows/go.yml/badge.svg)](https://github.com/gotray/got/actions/workflows/go.yml)
[![codecov](https://codecov.io/github/gotray/got/graph/badge.svg?token=V0ns2Rzmop)](https://codecov.io/github/gotray/got)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cpunion/go-python)
[![GitHub commits](https://badgen.net/github/commits/cpunion/go-python)](https://GitHub.com/Naereen/cpunion/go-python/commit/)
[![GitHub release](https://img.shields.io/github/v/tag/cpunion/go-python.svg?label=release)](https://github.com/gotray/got/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/gotray/got)](https://goreportcard.com/report/github.com/gotray/got)
[![Go Reference](https://pkg.go.dev/badge/github.com/gotray/got.svg)](https://pkg.go.dev/github.com/gotray/got)


Features:
- Don't need to install Go and Python, just need to download got
- Create a project with Go and Python environment configured
- Build, run, install Go packages with Python environment configured
- Compatible with Windows, Linux, MacOS

## Installation

```bash
go install github.com/gotray/got/cmd/got@latest
```

## Initialize a project

```bash
got init myproject
cd myproject
```

## Run project

```bash
got run .
```
