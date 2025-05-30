#!/bin/bash

sudo apt-get update
sudo apt-get --no-install-recommends -y install apt-transport-https ca-certificates wget

if [ ! -e /usr/share/keyrings/cznic-labs-pkg.gpg ]; then
    echo "Installing cznic-labs package signing key"
    sudo wget -O /usr/share/keyrings/cznic-labs-pkg.gpg https://pkg.labs.nic.cz/gpg
else
    echo "cznic-labs package signing key already installed"
fi

if [ ! -e /etc/apt/sources.list.d/cznic-labs.list ]; then
    echo "Adding cznic-labs package repository"
    echo "deb [signed-by=/usr/share/keyrings/cznic-labs-pkg.gpg] https://pkg.labs.nic.cz/bird3 bookworm main" | sudo tee /etc/apt/sources.list.d/cznic-labs.list

    echo "refreshing package list"
    sudo apt update
else
    echo "cznic-labs package repository already added"
fi

sudo apt-get --no-install-recommends -y install bird3

go install github.com/goreleaser/goreleaser/v2@latest
go install github.com/osrg/gobgp/v4/cmd/gobgp@latest
