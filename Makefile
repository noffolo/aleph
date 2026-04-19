.PHONY: build run frontend-dev nlp-dev build-models

build:
	cd frontend && npm run build
	go build -o aleph main.go

run: build
	./aleph

frontend-dev:
	cd frontend && npm run dev

nlp-dev:
	cd nlp && . venv/bin/activate && python3 main.py

build-models:
	cd nlp && . venv/bin/activate && python3 convert_onnx.py

clean:
	rm -rf dist aleph
	cd frontend && rm -rf node_modules dist
	cd nlp && rm -rf venv