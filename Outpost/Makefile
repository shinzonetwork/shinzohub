set-env:
	@echo "Setting environment variables"
	source secret.sh
	@echo "Environment variables set"

build:
	@echo "Building contract"
	forge build
	@echo "Contract built"

test:
	@echo "Testing contract"
	forge test
	@echo "Contract tested"

fmt:
	@echo "Formatting contract"
	forge fmt
	@echo "Contract formatted"

#    make greet MESSAGE="Hello, World!"
greet:
	@echo "Greeting"
	@echo "$(MESSAGE)"

# make install LIBRARY="<github_url_to_library>"
# Example: make install LIBRARY="https://github.com/OpenZeppelin/openzeppelin-contracts.git"
install:
	@echo "Installing dependencies"
	forge install $(LIBRARY)
	@echo "Dependencies installed"
