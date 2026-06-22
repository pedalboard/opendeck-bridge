BINARY = opendeck-bridge
REMOTE = laenzi@cm5-dev.home
REMOTE_DIR = /home/laenzi/projects/opendeck-bridge
UI_DIR = OpenDeckUI

.PHONY: build build-arm64 clean ui deploy restart

build:
	go build -o $(BINARY) .

build-arm64:
	CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ GOOS=linux GOARCH=arm64 go build -o $(BINARY)-arm64 .

ui:
	cd $(UI_DIR) && yarn && yarn build
	rm -rf ui
	cp -a $(UI_DIR)/dist ui

deploy: ui
	rsync -a --delete --exclude .git --exclude OpenDeckUI . $(REMOTE):$(REMOTE_DIR)/
	ssh $(REMOTE) "export PATH=\$$PATH:/usr/local/go/bin && cd $(REMOTE_DIR) && go build -o $(BINARY) . && pkill $(BINARY) || true"
	sleep 2
	ssh $(REMOTE) "cd $(REMOTE_DIR) && nohup ./$(BINARY) -port 'OpenDeck' -addr :8080 > /tmp/bridge.log 2>&1 &"
	@echo "Deployed. Logs: ssh $(REMOTE) tail -f /tmp/bridge.log"

restart:
	ssh $(REMOTE) "pkill $(BINARY) || true; sleep 1; cd $(REMOTE_DIR) && nohup ./$(BINARY) -port 'OpenDeck' -addr :8080 > /tmp/bridge.log 2>&1 &"

clean:
	rm -f $(BINARY) $(BINARY)-arm64
