#!/bin/bash
        if ! command -v kind >/dev/null 2>&1; then 
        echo "Installing kind..."
		echo "Detected architecture: $(uname -m)"
		if [ "$(uname -m)" = "x86_64" ]; then 
			if [ "$(uname -s)" = "Darwin" ]; then 
				curl -Lo ./kind/kind https://kind.sigs.k8s.io/dl/v0.31.0/kind-darwin-amd64; 
			else 
				curl -Lo ./kind/kind https://kind.sigs.k8s.io/dl/v0.31.0/kind-linux-amd64; 
			fi; 
		elif [ "$(uname -m)" = "aarch64" ]; then 
			curl -Lo ./kind/kind https://kind.sigs.k8s.io/dl/v0.31.0/kind-linux-arm64; 
		elif [ "$(uname -m)" = "arm64" ]; then 
			curl -Lo ./kind/kind https://kind.sigs.k8s.io/dl/v0.31.0/kind-darwin-arm64; 
		else 
			echo "Unsupported architecture: $(uname -m)"; 
			exit 1; 
		fi
		chmod +x ./kind/kind
		sudo mv ./kind/kind /usr/local/bin/kind
        else 
        echo "kind is already installed."
        fi
        