build:
	docker-compose build

run: build
	docker-compose up

logs:
	-docker-compose logs

init:
	rm -rf gen
	goa gen github.com/spuf/forward-basic-auth/design
	goa example github.com/spuf/forward-basic-auth/design
