BINARY := mdserve

.PHONY: run clean release

run: $(BINARY)
	./$(BINARY) .

$(BINARY): main.go
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

release:
	$(eval LAST := $(shell git tag --sort=-v:refname | head -1))
	$(eval VERSION := $(if $(LAST),v0.$(shell echo $(LAST) | sed 's/v0\.\([0-9]*\)\..*/\1/' | awk '{print $$1+1}').0,v0.1.0))
	git tag $(VERSION)
	git push origin $(VERSION)
	@echo "Tagged and pushed $(VERSION)"
