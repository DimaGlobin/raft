.PHONY: run clean

run:
	docker-compose up --build

clean:
	docker-compose down --rmi all --volumes --remove-orphans

build:
	docker-compose build
