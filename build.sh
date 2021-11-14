#!/bin/bash
if [ -z $1 ]
then
    imageName="resource-manager"
else    
    imageName=$1
fi

buildArgs=$(env | grep -iE "^http(s)?_proxy=.*$" | sed -r 's|^(.*)$|--build-arg \1|g' | tr '\n' ' ')

branch=$(git branch --show-current)

imageTag=$(echo ${imageName} | grep -oE ":\w+$")

if [ -z ${imageTag} ]
then
    if [ -z ${branch} ] || [ "${branch}" == "master" ]
    then
        tag=:latest
    else
        tag=:${branch}
    fi
fi

echo "=================="
echo "Start docker build"
echo "=================="
echo
echo "Run command: docker build ${buildArgs}-t ${imageName}${tag} ."
echo

docker build ${buildArgs}-t ${imageName}${tag} .