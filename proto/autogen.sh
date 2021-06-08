#!/bin/bash

cwd=$(dirname $0)

function absolute_path() {
	local _path=$1
	if [[ -d $_path ]]; then
		echo "$(cd $_path && pwd)"
	else
		echo $_path
	fi
}

function main() {
	local _cwd=$(absolute_path cwd)
}
