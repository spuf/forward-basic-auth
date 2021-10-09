build:
	docker-compose build

run: build
	docker-compose up

logs:
	-docker-compose logs
