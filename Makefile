.PHONY: build run frontend-dev nlp-dev build-models proto-python

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

proto-python:
	cd nlp && python3 -m grpc_tools.protoc -I../api/proto --python_out=. --grpc_python_out=. ../api/proto/aleph/nlp/v1/nlp.proto

clean:
	rm -rf dist aleph
	cd frontend && rm -rf node_modules dist
	cd nlp && rm -rf venv