#!/bin/bash

cp /okteto/.vscode-server/settings.json /.vscode-server/data/Machine
go get golang.org/x/tools/gopls@latest
go get github.com/go-delve/delve/cmd/dlv
