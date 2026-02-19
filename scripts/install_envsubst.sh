#!/bin/bash
if ! command -v envsubst >/dev/null 2>&1; then 
    echo "Installing envsubst..."
    if [ "$(uname -s)" = "Darwin" ]; then 
        brew install gettext; 
        brew link --force gettext; 
    elif [ "$(uname -s)" = "Linux" ]; then 
        if [ -f /etc/os-release ]; then 
            . /etc/os-release; 
            if [[ "$ID" == "ubuntu" || "$ID_LIKE" == *"debian"* ]]; then 
                sudo apt-get update && sudo apt-get install -y gettext-base; 
            elif [[ "$ID" == "fedora" || "$ID_LIKE" == *"rhel"* ]]; then 
                sudo dnf install -y gettext; 
            else 
                echo "Unsupported Linux distribution: $ID"; 
                exit 1; 
            fi; 
        else 
            echo "Cannot determine Linux distribution. Please install envsubst manually."; 
            exit 1; 
        fi;
    else 
        sudo apt-get update && sudo apt-get install -y gettext-base; 
    fi
else 
    echo "envsubst is already installed."
fi