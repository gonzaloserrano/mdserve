BINARY := mdserve

.PHONY: run install clean release

run: $(BINARY)
	./$(BINARY) .

$(BINARY): main.go
	go build -o $(BINARY) .

install:
	go install .

clean:
	rm -f $(BINARY)

release:
	$(eval LAST := $(shell git tag --sort=-v:refname | head -1))
	$(eval VERSION := $(if $(LAST),v0.$(shell echo $(LAST) | sed 's/v0\.\([0-9]*\)\..*/\1/' | awk '{print $$1+1}').0,v0.1.0))
	git tag $(VERSION)
	git push origin $(VERSION)
	@echo "Tagged and pushed $(VERSION)"
	@echo "Waiting for CI release..."
	@sleep 5 && gh run watch --exit-status
	gh release download $(VERSION) --pattern '*darwin*.zip' --dir dist
	@for zip in dist/*darwin*.zip; do \
		echo "Notarizing $$zip..."; \
		xcrun notarytool submit "$$zip" --keychain-profile "mdserve" --wait; \
	done
	@rm -rf dist
